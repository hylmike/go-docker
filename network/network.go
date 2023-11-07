package network

import (
	"crypto/rand"
	"fmt"
	"go-docker/utils"
	"log"
	mathRand "math/rand"
	"net"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const defaultBrige string = "br0"
const bridgeIpAddress = "172.29.0.1/16"

func IsDockerBridgeUp() (bool, error) {
	if links, err := netlink.LinkList(); err != nil {
		log.Printf("Failed to get list of network links\n")
		return false, err
	} else {
		for _, link := range links {
			if link.Type() == "bridge" && link.Attrs().Name == defaultBrige {
				return true, nil
			}
		}

		return false, err
	}
}

func SetupNetworkBridge() error {
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = defaultBrige
	dockerBridge := &netlink.Bridge{LinkAttrs: linkAttrs}

	if err := netlink.LinkAdd(dockerBridge); err != nil {
		return err
	}

	address, _ := netlink.ParseAddr(bridgeIpAddress)
	netlink.AddrAdd(dockerBridge, address)
	netlink.LinkSetUp(dockerBridge)
	return nil
}

func createMACAddress() net.HardwareAddr {
	hw := make(net.HardwareAddr, 6)
	hw[0] = 0x02
	hw[1] = 0x42
	rand.Read(hw[2:])

	return hw
}

func SetupVirtualEthOnHost(containerId string) error {
	veth0 := "veth0_" + containerId[:6]
	veth1 := "veth1_" + containerId[:6]
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = veth0
	veth0Struct := &netlink.Veth{
		LinkAttrs:        linkAttrs,
		PeerName:         veth1,
		PeerHardwareAddr: createMACAddress(),
	}

	if err := netlink.LinkAdd(veth0Struct); err != nil {
		return err
	}
	netlink.LinkSetUp(veth0Struct)
	dockerBridge, _ := netlink.LinkByName(defaultBrige)
	netlink.LinkSetMaster(veth0Struct, dockerBridge)

	return nil
}

func JoinContainerNetworkNamespace(containerId string) error {
	nsMountPath := utils.GetDockerNetNsPath() + "/" + containerId
	fd, err := unix.Open(nsMountPath, unix.O_RDONLY, 0)
	if err != nil {
		log.Printf("Failed to open network mount file for container %s: %v\n", containerId, err)
		return err
	}

	if err := unix.Setns(fd, unix.CLONE_NEWNET); err != nil {
		log.Printf("Failed to set new network for container %s: %v\n", containerId, err)
		return err
	}

	return nil
}

func SetupLocalInterface() {
	links, _ := netlink.LinkList()
	for _, link := range links {
		if link.Attrs().Name == "lo" {
			loAddr, _ := netlink.ParseAddr("127.0.0.1/32")
			if err := netlink.AddrAdd(link, loAddr); err != nil {
				log.Printf("Failed to configure local interface: %v\n", err)
			}
			netlink.LinkSetUp(link)
		}
	}
}

func SetupNewNetworkNamespace(containerId string) {
	_ = utils.CreateDirIfNotExist([]string{utils.GetDockerNetNsPath()})
	nsMount := utils.GetDockerNetNsPath() + "/" + containerId
	if _, err := unix.Open(nsMount, unix.O_RDONLY|unix.O_CREAT|unix.O_EXCL, 0644); err != nil {
		log.Fatalf("Failed to open mount file: %v\n", err)
	}

	fd, err := unix.Open("/proc/self/ns/net", unix.O_RDONLY, 0)
	defer unix.Close(fd)

	if err != nil {
		log.Fatalf("Failed to open net file: %v\n", err)
	}

	if err := unix.Unshare(unix.CLONE_NEWNET); err != nil {
		log.Fatalf("Failed to create unshared new net: %v\n", err)
	}
	if err := unix.Mount("/proc/self/ns/net", nsMount, "bind", unix.MS_BIND, ""); err != nil {
		log.Fatalf("Failed to mount network with file: %v\n", err)
	}
	if err := unix.Setns(fd, unix.CLONE_NEWNET); err != nil {
		log.Fatalf("Failed to set network with file: %v\n", err)
	}
}

func createPrivateIPAddress() string {
	byte1 := mathRand.Intn(254)
	byte2 := mathRand.Intn(254)

	return fmt.Sprintf("172,29,%d.%d", byte1, byte2)
}

func SetupContainerNetworkInterface(containerId string) {
	nsMount := utils.GetDockerNetNsPath() + "/" + containerId
	fd, err := unix.Open(nsMount, unix.O_RDONLY, 0)
	defer unix.Close(fd)

	if err != nil {
		log.Fatalf("Failed to open network mount file: %v\n", err)
	}

	//Set container veth1 to new network namespace
	veth1 := "veth1" + containerId[:6]
	veth1Link, err := netlink.LinkByName(veth1)
	if err != nil {
		log.Fatalf("Failed to fetch veth1: %v\n", err)
	}
	if err := netlink.LinkSetNsFd(veth1Link, fd); err != nil {
		log.Fatalf("Failed to set network namespace for veth1: %v\n", err)
	}

	if err := unix.Setns(fd, unix.CLONE_NEWNET); err != nil {
		log.Fatalf("Failed to set network: %v\n", err)
	}

	addr, _ := netlink.ParseAddr(createPrivateIPAddress() + "/16")
	if err := netlink.AddrAdd(veth1Link, addr); err != nil {
		log.Fatalf("Failed to assign IP to veth1: %v\n", err)
	}

	//Activate veth1 interface
	if err := netlink.LinkSetUp(veth1Link); err != nil {
		log.Fatalf("Failed to activate veth1: %v\n", err)
	}

	//Add a default route
	const defaultGateway = "172.29.0.1"
	route := netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: veth1Link.Attrs().Index,
		Gw:        net.ParseIP(defaultGateway),
		Dst:       nil,
	}

	if err := netlink.RouteAdd(&route); err != nil {
		log.Fatalf("Failed to add default route: %v\n", err)
	}
}
