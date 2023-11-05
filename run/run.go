package run

import (
	"crypto/rand"
	"fmt"
	"go-docker/image"
	"go-docker/network"
	"go-docker/utils"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func createContainerId() string {
	randBytes := make([]byte, 6)
	rand.Read(randBytes)

	return fmt.Sprintf(
		"%02x%02x%02x%02x%02x%02x",
		randBytes[0],
		randBytes[1],
		randBytes[2],
		randBytes[3],
		randBytes[4],
		randBytes[5],
	)
}

func createContainerDirs(containerId string) {
	containerHome := utils.GetDockerContainerPath() + "/" + containerId
	containerDirs := []string{
		containerHome + "/fs",
		containerHome + "/fs/mnt",
		containerHome + "/fs/upperdir",
		containerHome + "/fs/workdir",
	}

	if err := utils.CreateDirIfNotExist(containerDirs); err != nil {
		log.Fatalf("Failed to create container directories: %v\n", err)
	}
}

func getContainerFSHome(containerId string) string {
	return utils.GetDockerContainerPath() + "/" + containerId + "/fs"
}

func mountOveryFileSystem(containerId string, imgShaHex string) {
	var srcLayers []string
	pathManifest := image.GetManifestPathForImage(imgShaHex)
	mf := utils.Manifest{}
	utils.ParseManifest(pathManifest, &mf)

	if len(mf) == 0 || len(mf[0].Layers) == 0 {
		log.Fatal("Can't find any layers")
	}
	if len(mf) > 1 {
		log.Fatal("Can't handle more than one manifest")
	}

	imageBasePath := image.GetBasePathForImage(imgShaHex)
	for _, layer := range mf[0].Layers {
		srcLayers = append(srcLayers, imageBasePath+"/"+layer[:12]+"/fs")
	}

	containerFSHome := getContainerFSHome(containerId)
	montOptions := "lowerdir=" + strings.Join(srcLayers, ":") + ",upperdir=" + containerFSHome + "/upperdir, workdir=" + containerFSHome + "workdir"
	if err := unix.Mount("none", containerFSHome+"/mnt", "overlay", 0, montOptions); err != nil {
		log.Fatalf("Mount overlay file system failed: %v\n", err)
	}
}

func prepareAndExecuteContainer(
	mem int,
	swap int,
	pids int,
	cpus float64,
	containerId string,
	imgShaHex string,
	cmdArgs []string,
) {
	//Setup network namaspace
	cmd := &exec.Cmd{
		Path:   "proc/self/exe",
		Args:   []string{"/proc/self/exe", "setup-netns", containerId},
		Stdout: os.Stdout,
		Stdin:  os.Stdin,
	}
	cmd.Run()

	//Setup virtual ethernet inferface
	cmd = &exec.Cmd{
		Path:   "proc/self/exe",
		Args:   []string{"/proc/self/exe", "setup-veth", containerId},
		Stdout: os.Stdout,
		Stdin:  os.Stdin,
	}
	cmd.Run()

	//Setup resource limitation
	var options []string
	if mem > 0 {
		options = append(options, "--mem="+strconv.Itoa(mem))
	}
	if swap > 0 {
		options = append(options, "--swap="+strconv.Itoa(swap))
	}
	if pids > 0 {
		options = append(options, "--pids="+strconv.Itoa(pids))
	}
	if cpus > 0 {
		options = append(options, "--cpus="+strconv.FormatFloat(cpus, 'f', 1, 64))
	}
	options = append(options, "--img="+imgShaHex)

	args := append([]string{containerId}, cmdArgs...)
	args = append(options, args...)
	args = append([]string{"child-mode"}, args...)
	cmd = exec.Command("/proc/self/exe", args...)
	cmd.SysProcAttr = &unix.SysProcAttr{
		/*
			From namespaces(7)
				Namespace Flag            Isolates
				--------- ----   		 --------
				Cgroup    CLONE_NEWCGROUP Cgroup root directory
				IPC       CLONE_NEWIPC    System V IPC,
											POSIX message queues
				Network   CLONE_NEWNET    Network devices,
											stacks, ports, etc.
				Mount     CLONE_NEWNS     Mount points
				PID       CLONE_NEWPID    Process IDs
				Time      CLONE_NEWTIME   Boot and monotonic
											clocks
				User      CLONE_NEWUSER   User and group IDs
				UTS       CLONE_NEWUTS    Hostname and NIS
											domain name
		*/
		Cloneflags: unix.CLONE_NEWIPC | unix.CLONE_NEWNS | unix.CLONE_NEWPID | unix.CLONE_NEWUTS,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to create container %s: %v\n", containerId, err)
	}

}

func InitContainer(mem int, swap int, pids int, cpus float64, src string, options []string) {
	containerId := createContainerId()
	log.Printf("New container ID: %s\n", containerId)
	imageShaHex := image.DownloadImageIfRequired(src)

	log.Printf("Image to overlay mount: %s\n", imageShaHex)
	createContainerDirs(containerId)
	mountOveryFileSystem(containerId, imageShaHex)

	if err := network.SetupVirtualEthOnHost(containerId); err != nil {
		log.Fatalf("Failed to setup Veth0 on host: %v", err)
	}

	prepareAndExecuteContainer(mem, swap, pids, cpus, containerId, imageShaHex, options)
	log.Println("Container setup in finished!")
}

func unmountNetworkNamespace(containerId string) {
	netNsPath := utils.GetDockerNetNsPath() + "/" + containerId
	if err := unix.Unmount(netNsPath, 0); err != nil {
		log.Fatalf("Failed to unmount network namespace %s: %v\n", netNsPath, err)
	}
}

func unmountContainerFileSystem(containerId string) {
	mountFSPath := utils.GetDockerContainerPath() + "/" + containerId + "/fs/mnt"
	if err := unix.Unmount(mountFSPath, 0); err != nil {
		log.Fatalf("Failed to unmount container file system %s: %v\n", mountFSPath, err)
	}
}

func removeCGroups(containerId string) {
	cgroupDirs := []string{
		"/sys/fs/cgroup/memory/go-docker/" + containerId,
		"/sys/fs/cgroup/pids/go-docker/" + containerId,
		"/sys/fs/cgroup/cpu/go-docker/" + containerId,
	}

	for _, cgroupDir := range cgroupDirs {
		if err := os.Remove(cgroupDir); err != nil {
			log.Fatalf("Failed to remove cgroup dir %s: %v\n", cgroupDir, err)
		}
	}
}

func removeContainerDirs(containerId string) {
	containerDir := utils.GetDockerContainerPath() + "/" + containerId
	if err := os.RemoveAll(containerDir); err != nil {
		log.Fatalf("Failed to remove container directory: %v", err)
	}
}

func CleanUpContainer(containerId string) {
	containerPath := utils.GetDockerContainerPath() + "/" + containerId
	if _, err := os.Stat(containerPath); os.IsNotExist(err) {
		log.Fatalf("Invalid container id %s", containerId)
	}

	unmountNetworkNamespace(containerId)
	unmountContainerFileSystem(containerId)
	removeCGroups(containerId)
	removeContainerDirs(containerId)
}
