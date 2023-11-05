package ps

import (
	"bufio"
	"fmt"
	"go-docker/image"
	"go-docker/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ContainerInfo struct {
	containerId string
	image       string
	command     string
	pid         int
}

const basePath string = "sys/fs/cgroup/cpu/go-docker"

func getDistribution(containerId string) (string, error) {
	var lines []string
	file, err := os.Open("/proc/mounts")
	if err != nil {
		fmt.Printf("Failed to read /proc/mounts: %v\n", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for _, line := range lines {
		if strings.Contains(line, containerId) {
			layers := strings.Split(line, " ")
			for _, layer := range layers {
				options := strings.Split(layer, ",")
				for _, option := range options {
					if strings.Contains(option, "lowerdir=") {
						imagesPath := utils.GetDockerImagePath()
						leadString := "lowerdir=" + imagesPath + "/"
						trailerString := option[len(leadString):]
						imageId := trailerString[:12]
						imageName, tag := image.GetImageNameAndTag(imageId)

						return fmt.Sprintf("%s:%s", imageName, tag), nil
					}
				}
			}
		}
	}

	return "", nil
}

func getContainerDetailsForId(containerId string) (ContainerInfo, error) {
	container := ContainerInfo{}
	var procs []string
	procsPath := basePath + "/" + containerId + "/cgroup.procs"

	file, err := os.Open(procsPath)
	if err != nil {
		fmt.Println("Failed to read cgroup procs")
		return container, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		procs = append(procs, scanner.Text())
	}

	if len(procs) > 0 {
		pid, err := strconv.Atoi(procs[len(procs)-1])
		if err != nil {
			fmt.Printf("Failed to read PID of container %s: %v\n", containerId, err)
		}
		cmd, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
		if err != nil {
			fmt.Println("Failed to read command link")
			return container, err
		}

		containerMountPath := utils.GetDockerContainerPath() + "/" + containerId + "/fs/mnt"
		realContainerMountPath, err := filepath.EvalSymlinks(containerMountPath)
		if err != nil {
			fmt.Println("Failed to resolve container mounting path")
			return container, err
		}

		image, _ := getDistribution(containerId)
		container = ContainerInfo{
			containerId: containerId,
			image:       image,
			command:     cmd[len(realContainerMountPath):],
			pid:         pid,
		}
	}

	return container, nil
}

func GetRunningContainers() ([]ContainerInfo, error) {
	var containers []ContainerInfo
	basePath := utils.GetDockerContainerPath()

	entries, err := os.ReadDir(basePath)

	if os.IsNotExist(err) {
		return containers, nil
	} else {
		if err != nil {
			return nil, err
		} else {
			for _, entry := range entries {
				if entry.IsDir() {
					container, _ := getContainerDetailsForId(entry.Name())
					if container.pid > 0 {
						containers = append(containers, container)
					}
				}
			}

			return containers, nil
		}
	}
}

func PrintRunningContainers() {
	containers, err := GetRunningContainers()
	if err != nil {
		os.Exit(1)
	}

	fmt.Println("CONTAINER ID\tIMAGE\tCOMMAND")
	for _, container := range containers {
		fmt.Printf("%s\t%s\t%s\n", container.containerId, container.image, container.command)
	}
}
