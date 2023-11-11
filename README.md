# Go-docker

## Main Capabilities
Go-docker can emulate the core of Docker functions, letting you manage Docker images (get from Docker Hub), run containers, list running containers or execute a process in an already running container:
- Same as docker, go-docker uses namsspace to isolate environment and resource
- Same as docker, go-docker uses control group to limited resources (cpu, memory, pids) used by new created containers
- Same as docker, go-docker uses overlay file system to quickly create containers without copying whole file system while also sharing the same container image between multiple container instances.


Following are major command supported by go-docker
* Run a process in a container
   * `go-docker run <--cpus=cpus-max> <--mem=mem-max> <--pids=pids-max> <image[:tag]> </path/to/command>`
* List running / active containers
   * `go-docker ps`
* List all containers, including running ones and inactive ones
   * `go-docker containers` 
* Activate container
   * `go-docker actContainer <containerId>`
* Run command inside a container with id
   * `go-docker exec <containerId> <command>`
* List all the local images
   * `go-docker images`
* Clean a container and related data with id
   * `go-docker rmContainer <containerId>`
* Delete a local image and related metadata with id
   * `go-docker rmImage <imageId>`

Following are some examples of usage:
```
/go-docker$ sudo ./go-docker run alpine /bin/sh
2023/11/09 22:46:02 New container ID: 888118c4a5e8
2023/11/09 22:46:02 Download metadata for alpine:latest, please wait...
2023/11/09 22:46:02 image hash: 8ca4688f4f35
2023/11/09 22:46:02 Checking if image exists under with other names
2023/11/09 22:46:02 Image don't exist. Downloading...
2023/11/09 22:46:03 Success downloaded alpine
2023/11/09 22:46:03 Umcompressing layer to /var/lib/go-docker/images/8ca4688f4f35/96526aa774ef/fs
2023/11/09 22:46:03 Image to overlay mount: 8ca4688f4f35
/ # ps aux
PID   USER     TIME  COMMAND
    1 root      0:00 /proc/self/exe inner-mode 888118c4a5e8 /bin/sh
    7 root      0:00 /bin/sh
    8 root      0:00 ps aux
/ # ifconfig
lo        Link encap:Local Loopback  
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000 
          RX bytes:0 (0.0 B)  TX bytes:0 (0.0 B)

/ # ls -la
total 44
drwxr-xr-x    1 root     root            80 Nov 10 03:46 .
drwxr-xr-x    1 root     root            80 Nov 10 03:46 ..
drwxr-xr-x    2 root     root          4096 Nov 10 03:46 bin
drwxrwxrwt    3 root     root            60 Nov 10 03:46 dev
drwxr-xr-x    1 root     root            60 Nov 10 03:46 etc
drwxr-xr-x    2 root     root          4096 Nov 10 03:46 home
drwxr-xr-x    7 root     root          4096 Nov 10 03:46 lib
drwxr-xr-x    5 root     root          4096 Nov 10 03:46 media
drwxr-xr-x    2 root     root          4096 Nov 10 03:46 mnt
drwxr-xr-x    2 root     root          4096 Nov 10 03:46 opt
dr-xr-xr-x  540 root     root             0 Nov 10 03:46 proc
drwx------    1 root     root            60 Nov 10 03:46 root
drwxr-xr-x    2 root     root          4096 Nov 10 03:46 run
drwxr-xr-x    2 root     root          4096 Nov 10 03:46 sbin
drwxr-xr-x    2 root     root          4096 Nov 10 03:46 srv
dr-xr-xr-x   13 root     root             0 Nov 10 03:46 sys
drwxrwxrwt    2 root     root            40 Nov 10 03:46 tmp
drwxr-xr-x    7 root     root          4096 Nov 10 03:46 usr
drwxr-xr-x   12 root     root          4096 Nov 10 03:46 var

/go-docker$ sudo ./go-docker ps
[sudo] password for michael: 
CONTAINER ID    IMAGE   COMMAND
d66f2a7efe90    alpine:latest   /bin/busybox

/go-docker$ sudo ./go-docker images
Image   Tag     ID
alpine
                  latest        8ca4688f4f35
ubuntu
                  latest        e4c58958181a

/go-docker$ sudo ./go-docker rmImage e4c58958181a                  
```

The tool and major functions has been tested with Ubuntu 22.x and 23.x with latest go libraries, details can refer to go.mod file

Reference: This tool heavily refer to gocker (https://github.com/shuveb/containers-the-hard-way), but rewrote some parts with latest librarys and linux configurations, fixed bugs also with better file structure and error handling design