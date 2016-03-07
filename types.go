package client

import (
	"github.com/samalba/dockerclient"
)

type VirtContainerConfig struct {
	Name            string `json:"name"`
	Memory          int64  `json:"memory"`
	Vcpu            int64  `json:"vcpu"`
	Disk_source     string `json:"disk_source"`
	Cdrom_source    string `json:"cdrom_source"`
	Bridge          string `json:"bridge"`
	Boot            string `json:"boot"`
	ContainerConfig *dockerclient.ContainerConfig
}

type VirtContainerInfo struct {
	Id              string
	Name            string
	CpuInfo         *VirtCpuInfo
	DomInfo         *VirtDomInfo
	ContainerConfig *dockerclient.ContainerConfig
}

type VirtCpuInfo struct {
	CpuTime  int64 `json:"cpu_time"`
	SysTime  int64 `json:"system_time"`
	UserTime int64 `json:"user_time"`
}

type VirtDomInfo struct {
	Status  int64 `json:"status"`
	UsedMem int64 `json:"usedMemory"`
	MaxMem  int64 `json:"maxMemory"`
	CpuTime int64 `json:"cpuTime"`
	VirtCpu int64 `json:"virtCpu"`
}
