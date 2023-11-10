# Go-docker

## Main Capabilities
Go-docker can emulate the core of Docker functions, letting you manage Docker images (get from Docker Hub), run containers, list running containers or execute a process in an already running container:
- Same as docker, go-docker uses namsspace to isolate environment and resource
- Same as docker, go-docker uses control group to limited resources (cpu, memory, pids) used by new created containers
- Same as docker, go-docker uses overlay file system to quickly create containers without copying whole file system while also sharing the same container image between multiple container instances.


Following are major command supported by go-docker
* Run a process in a container
   * `go-docker run <--cpus=cpus-max> <--mem=mem-max> <--pids=pids-max> <image[:tag]> </path/to/command>`
* List running containers
   * `go-docker ps`
* Run command inside a container with id
   * `go-docker exec <containerId> <command>`
* List all the local images
   * `go-docker images`
* Clean a container and related data with id
   * `go-docker clean <containerId>`
* Delete a local image and related metadata with id
   * `go-docker rmImage <imageId>`

Following are some examples of usage:


The tool and major functions has been tested with Ubuntu 22.x and 23.x with latest go libraries, details can refer to go.mod file

Reference: This tool heavily refer to gocker (https://github.com/shuveb/containers-the-hard-way), but rewrote some parts with latest librarys and linux configurations, fixed bugs also with better file structure and error handling design