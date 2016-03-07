package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/samalba/dockerclient"
)

var (
	ErrImageNotFound = errors.New("Image not found")
	ErrNotFound      = errors.New("Not found")
	defaultTimeout   = 30 * time.Second
)

type LibvirtplusClient struct {
	URL           *url.URL
	HTTPClient    *http.Client
	TLSConfig     *tls.Config
	monitorStats  int32
	eventStopChan chan (struct{})
}

type Error struct {
	StatusCode int
	Status     string
	msg        string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Status, e.msg)
}

func NewLibvirtplusClient(daemonUrl string, tlsConfig *tls.Config) (*LibvirtplusClient, error) {
	return NewLibvirtplusClientTimeout(daemonUrl, tlsConfig, time.Duration(defaultTimeout))
}

func NewLibvirtplusClientTimeout(daemonUrl string, tlsConfig *tls.Config, timeout time.Duration) (*LibvirtplusClient, error) {
	u, err := url.Parse(daemonUrl)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Scheme == "tcp" {
		if tlsConfig == nil {
			u.Scheme = "http"
		} else {
			u.Scheme = "https"
		}
	}
	httpClient := newHTTPClient(u, tlsConfig, timeout)
	return &LibvirtplusClient{u, httpClient, tlsConfig, 0, nil}, nil
}

func (client *LibvirtplusClient) doRequest(method string, path string, body []byte, headers map[string]string) ([]byte, error) {
	b := bytes.NewBuffer(body)

	reader, err := client.doStreamRequest(method, path, b, headers)
	if err != nil {
		return nil, err
	}

	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (client *LibvirtplusClient) doStreamRequest(method string, path string, in io.Reader, headers map[string]string) (io.ReadCloser, error) {
	if (method == "POST" || method == "PUT") && in == nil {
		in = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, client.URL.String()+path, in)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	if headers != nil {
		for header, value := range headers {
			req.Header.Add(header, value)
		}
	}
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		if !strings.Contains(err.Error(), "connection refused") && client.TLSConfig == nil {
			return nil, fmt.Errorf("%v. Are you trying to connect to a TLS-enabled daemon without TLS?", err)
		}
		return nil, err
	}
	if resp.StatusCode == 404 {
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, ErrNotFound
		}
		if len(data) > 0 {
			// check if is image not found error
			if strings.Index(string(data), "No such image") != -1 {
				return nil, ErrImageNotFound
			}
			return nil, errors.New(string(data))
		}
		return nil, ErrNotFound
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, Error{StatusCode: resp.StatusCode, Status: resp.Status, msg: string(data)}
	}
	return resp.Body, nil
}

func (client *LibvirtplusClient) ListContainers() ([]dockerclient.Container, error) {
	uri := fmt.Sprintf("/containers")

	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}

	var ids []string
	err = json.Unmarshal(data, &ids)
	if err != nil {
		return nil, err
	}
	fmt.Println(ids)
	ret := []dockerclient.Container{}

	for _, id := range ids {
		containerInfo, err := client.InspectContainer(id)
		if err != nil {
			fmt.Println(err)
			continue
		}
		container := &dockerclient.Container{
			Id:    containerInfo.Id,
			Names: []string{containerInfo.Name},
			Image: containerInfo.Image,
		}
		if containerInfo.State.Running == true {
			container.Status = "Running"
		} else {
			container.Status = "Not Running"
		}
		ret = append(ret, *container)
	}
	return ret, nil
}

func (client *LibvirtplusClient) InspectContainer(id string) (*dockerclient.ContainerInfo, error) {
	uri := fmt.Sprintf("/containers/%s", id)
	data, err := client.doRequest("GET", uri, nil, nil)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(data))
	virtInfo := &VirtContainerInfo{}
	err = json.Unmarshal(data, virtInfo)
	if err != nil {
		return nil, err
	}
	info := &dockerclient.ContainerInfo{
		Id:     virtInfo.Id,
		Name:   virtInfo.Name,
		Config: virtInfo.ContainerConfig,
	}
	if virtInfo.ContainerConfig != nil {
		info.Image = virtInfo.ContainerConfig.Image
	}
	if virtInfo.DomInfo.Status == 1 {
		info.State = &dockerclient.State{
			Running: true,
		}
	}
	return info, nil
}

func (client *LibvirtplusClient) CreateContainer(ccf *dockerclient.ContainerConfig, name string) (string, error) {
	config := VirtContainerConfig{
		Name:            name,
		Memory:          ccf.HostConfig.Memory,
		Vcpu:            ccf.HostConfig.CpuQuota,
		Bridge:          ccf.HostConfig.NetworkMode,
		Disk_source:     "",
		Cdrom_source:    "",
		Boot:            ccf.Cmd[0],
		ContainerConfig: ccf,
	}
	if ccf.Cmd[0] == "hd" {
		config.Disk_source = ccf.Image
	} else if ccf.Cmd[0] == "cdrom" {
		config.Cdrom_source = ccf.Image
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	uri := fmt.Sprintf("/containers")
	if name != "" {
		v := url.Values{}
		v.Set("name", name)
		uri = fmt.Sprintf("%s?%s", uri, v.Encode())
	}
	data, err = client.doRequest("POST", uri, data, nil)
	if err != nil {
		return "", err
	}
	id := ""
	err = json.Unmarshal(data, id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (client *LibvirtplusClient) RemoveContainer(id string) error {
	uri := fmt.Sprintf("/containers/%s", id)
	_, err := client.doRequest("DELETE", uri, nil, nil)
	return err
}
