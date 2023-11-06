# Go-docker

## Main Capabilities
Go-docker can emulate the core of Docker functions, letting you manage Docker images (which it gets from Docker Hub), run containers, list running containers or execute a process in an already running container:
* Run a process in a container
   * `go-docker run <--cpus=cpus-max> <--mem=mem-max> <--pids=pids-max> <image[:tag]> </path/to/command>`
* List running containers
   * `go-docker ps`
* Clean a container with id
   * `go-docker clean <containerId>`