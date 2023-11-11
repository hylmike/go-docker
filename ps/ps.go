package ps

import (
	"bufio"
	"fmt"
	"go-docker/image"
	"go-docker/utils"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ContainerInfo struct {
	ContainerId string
	Image       string
	Status      string
	Command     string
	Pid         int
}

const basePath string = "/sys/fs/cgroup/cpu/go-docker"

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
						imageName, tag := image.ImageExistByHash(imageId)

						return fmt.Sprintf("%s:%s", imageName, tag), nil
					}
				}
			}
		}
	}

	return "", nil
}

func GetContainerDetailsForId(containerId string) (ContainerInfo, error) {
	image, _ := getDistribution(containerId)
	container := ContainerInfo{
		ContainerId: containerId,
		Image:       image,
	}

	var procs []string
	procsPath := basePath + "/" + containerId + "/cgroup.procs"

	file, err := os.Open(procsPath)
	if err != nil {
		fmt.Printf("Failed to read cgroup procs: %v\n", err)
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

		container.Command = cmd[len(realContainerMountPath):]
		container.Pid = pid
	}

	return container, nil
}

func GetContainers(activeOnly bool) ([]ContainerInfo, error) {
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
					container, _ := GetContainerDetailsForId(entry.Name())
					if activeOnly {
						if container.Pid > 0 {
							container.Status = "active"
							containers = append(containers, container)
						}
					} else {
						if container.Pid > 0 {
							container.Status = "active"
						} else {
							container.Status = "inactive"
						}
						containers = append(containers, container)
					}
				}
			}

			return containers, nil
		}
	}
}

func PrintRunningContainers() {
	containers, err := GetContainers(true)
	if err != nil {
		os.Exit(1)
	}

	fmt.Println("CONTAINER ID\tIMAGE\t\tCOMMAND")
	for _, container := range containers {
		fmt.Printf("%s\t%s\t%s\n", container.ContainerId, container.Image, container.Command)
	}
}

func RemoveImageByHash(imgShaHex string) {
	imageName, tag := image.ImageExistByHash(imgShaHex)
	if imageName == "" {
		log.Fatalf("Can't find image %s\n", imgShaHex)
	}

	containers, err := GetContainers(true)
	if err != nil {
		log.Fatalf("Failed to get running container list: %v\n", err)
	}

	for _, container := range containers {
		if container.Image == imageName+":"+tag {
			log.Fatalf("Can't remove this image as it is used by container %s\n", container.ContainerId)
		}
	}

	if err := os.RemoveAll(utils.GetDockerImagePath() + "/" + imgShaHex); err != nil {
		log.Fatalf("Failed to remvoe image directory: %v\n", err)
	}

	image.RemoveImageMetadata(imgShaHex)
}

func PrintAllContainers() {
	containers, err := GetContainers(false)
	if err != nil {
		log.Fatalf("Failed to load containers: %v\n", err)
	}

	fmt.Println("CONTAINER ID\tIMAGE\t\tSTATUS")
	for _, container := range containers {
		fmt.Printf("%s\t%s\t%s\n", container.ContainerId, container.Image, container.Status)
	}
}
