package cgroups

import (
	"fmt"
	"go-docker/utils"
	"log"
	"os"
	"runtime"
	"strconv"
)

func getMemBaseDir(containerId string) string {
	return "/sys/fs/cgroup/memory/go-docker/" + containerId
}

func getPidsBaseDir(containerId string) string {
	return "/sys/fs/cgroup/pids/go-docker/" + containerId
}

func getCpuBaseDir(containerId string) string {
	return "/sys/fs/cgroup/cpu/go-docker/" + containerId
}

func getCGroupDirc(containerId string) []string {
	return []string{
		getMemBaseDir(containerId),
		getPidsBaseDir(containerId),
		getCpuBaseDir(containerId),
	}
}

func CreateCGroup(containerId string, createCGroupDirs bool) {
	cgroupDirs := getCGroupDirc(containerId)

	if createCGroupDirs {
		if err := utils.CreateDirIfNotExist(cgroupDirs); err != nil {
			log.Fatalf("Failed to creatre directories for cgroup: %v\n", err)
		}
	}

	for _, cgroupDir := range cgroupDirs {
		if err := os.WriteFile(cgroupDir+"/notify_on_release", []byte("1"), 0700); err != nil {
			log.Fatalf("Failed to write to cgroup notification file: %v\n", err)
		}

		if err := os.WriteFile(cgroupDir+"/cgroup.procs", []byte(strconv.Itoa(os.Getpid())), 0700); err != nil {
			log.Fatalf("Failed to write to cgroup procs file")
		}
	}
}

func RemoveCGroups(containerId string) {
	cgroupDirs := getCGroupDirc(containerId)

	for _, cgroupDir := range cgroupDirs {
		if err := os.Remove(cgroupDir); err != nil {
			log.Fatalf("Failed to remove cgroup dir %s: %v\n", cgroupDir, err)
		}
	}
}

func setMemoryLimit(containerId string, limitMB int, swapLimitMB int) {
	memFilePath := getMemBaseDir(containerId) + "/memory.limit_in_bytes"
	swapFilePath := getMemBaseDir(containerId) + "/memory.memsw.limit_in_bytes"

	if err := os.WriteFile(memFilePath, []byte(strconv.Itoa(limitMB*1024*1024)), 0644); err != nil {
		log.Fatalf("Failed to write memory limit: %v\n", err)
	}

	if swapLimitMB >= 0 {
		if err := os.WriteFile(swapFilePath, []byte(strconv.Itoa((limitMB*1024*1024)+(swapLimitMB*1024*1024))), 0644); err != nil {
			log.Fatalf("Failed to write swap memory limit: %v\n", err)
		}
	}
}

func setCpuLimit(containerId string, limit float64) {
	cfsPeriodPath := getCpuBaseDir(containerId) + "/cpu.cfs_period_us"
	cfsQuotaPath := getCpuBaseDir(containerId) + "/cpu.cfs_quota_us"

	if limit > float64(runtime.NumCPU()) {
		fmt.Println("Ignore attempt to config CPU quota more than available CPUs")
		return
	}

	if err := os.WriteFile(cfsPeriodPath, []byte(strconv.Itoa(1000000)), 0644); err != nil {
		log.Fatalf("Failed to write CFS period: %v\n", err)
	}

	if err := os.WriteFile(cfsQuotaPath, []byte(strconv.Itoa(int(1000000*limit))), 0644); err != nil {
		log.Fatalf("Failed to write CFS quota: %v\n", err)
	}
}

func setPidsLimit(containerId string, limit int) {
	maxProcPath := getPidsBaseDir(containerId) + "/pids.max"

	if err := os.WriteFile(maxProcPath, []byte(strconv.Itoa(limit)), 0644); err != nil {
		log.Fatalf("Failed to write pids limit: %v\n", err)
	}
}

func ConfigCGroup(containerId string, mem int, swap int, pids int, cpus float64) {
	if mem > 0 {
		setMemoryLimit(containerId, mem, swap)
	}
	if cpus > 0 {
		setCpuLimit(containerId, cpus)
	}
	if pids > 0 {
		setPidsLimit(containerId, pids)
	}
}
