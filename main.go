package main

import (
	"flag"
	"fmt"
	"go-docker/image"
	"go-docker/network"
	"go-docker/ps"
	"go-docker/run"
	"go-docker/utils"
	"log"
	"os"
)

func main() {
	command := os.Args[1]
	if len(os.Args) < 2 || !utils.ValidCommand(command) {
		utils.ShowGuide()
		os.Exit(1)
	}

	//Need root privileges to write into system directory
	if os.Getegid() != 0 {
		log.Fatal("You need root privileges to run this tool")
	}

	//Create directories we need
	if err := utils.InitDockerDirs(); err != nil {
		log.Fatalf("Failed to create basic directories for tool: %v", err)
	}

	switch command {
	case "run":
		flags := flag.FlagSet{}
		mem := flags.Int("mem", -1, "Max RAM to allow in MB")
		swap := flags.Int("swap", -1, "Max swap to allow in MB")
		pids := flags.Int("pids", -1, "Number of max processes to allow")
		cpus := flags.Float64("cpus", -1, "Number of CPU cores to allow to use")

		if err := flags.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing input parameters: ", err)
		}
		if len(flags.Args()) < 2 {
			log.Fatal("Please pass image name and command to run")
		}

		//Create network bridge br0 required in tool
		if isOn, _ := network.IsDockerBridgeUp(); !isOn {
			log.Printf("Setup and turn on the default network bridge...")
			if err := network.SetupNetworkBridge(); err != nil {
				log.Fatalf("Failed to create default network bridge: %v", err)
			}
		}

		//Initialize the container based on inputs
		run.InitContainer(*mem, *swap, *pids, *cpus, flags.Args()[0], flags.Args()[1:])
	case "inner-mode":
		//Inside container mode, to run command inside container
		flags := flag.FlagSet{}
		mem := flags.Int("mem", -1, "Max RAM to allow in MB")
		swap := flags.Int("swap", -1, "Max swap to allow in MB")
		pids := flags.Int("pids", -1, "Number of max processes to allow")
		cpus := flags.Float64("cpus", -1, "Number of CPU cores to allow to use")

		if err := flags.Parse(os.Args[2:]); err != nil {
			fmt.Printf("Failed to parse input flags: %v\n", err)
		}
		if len(flags.Args()) < 2 {
			log.Fatalln("Need image name and command to run inside container")
		}
		run.SetupContainerExecCommand(*mem, *swap, *pids, *cpus, flags.Args()[0], flags.Args()[1:])
	case "ps":
		ps.PrintRunningContainers()
	case "setup-netns":
		network.SetupNewNetworkNamespace(os.Args[2])
	case "setup-veth":
		network.SetupContainerNetworkInterface(os.Args[2])
	case "exec":
		run.ExecCommandInContainer(os.Args[2])
	case "images":
		image.PrintImages()
	case "clean":
		if len(os.Args) < 3 {
			utils.ShowGuide()
			os.Exit(1)
		}
		run.CleanUpContainer(os.Args[2])
	case "rmImage":
		if len(os.Args) < 3 {
			utils.ShowGuide()
			os.Exit(1)
		}
		ps.RemoveImageByHash(os.Args[2])
	default:
		utils.ShowGuide()
	}
}
