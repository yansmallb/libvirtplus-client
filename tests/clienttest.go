package main

import (
	"fmt"
	"github.com/samalba/dockerclient"
	client "github.com/yansmallb/libvirtplus-client"
)

func main() {
	c, err := client.NewLibvirtplusClient("192.168.11.51:2376", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	list, err := TestList(c)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(list)

	if true {
		err := TestDelete(c, "192.168.11.51_13")
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		id, err := TestCreate(c)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(id)
	}
}

func TestCreate(c *client.LibvirtplusClient) (string, error) {
	config := &dockerclient.ContainerConfig{
		Cmd:   []string{"hd"},
		Image: "/var/lib/libvirt/images/centos_65.qcow2",
		HostConfig: dockerclient.HostConfig{
			CpuQuota:    4,
			NetworkMode: "virbr0",
			Memory:      1048576,
		},
	}
	return c.CreateContainer(config, "centos_65_3")
}

func TestList(c *client.LibvirtplusClient) ([]dockerclient.Container, error) {
	list, err := c.ListContainers()
	return list, err
}

func TestDelete(c *client.LibvirtplusClient, id string) error {
	return c.RemoveContainer(id)
}
