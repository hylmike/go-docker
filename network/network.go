package network

import (
	"crypto/rand"
	"log"
	"net"

	"github.com/vishvananda/netlink"
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
