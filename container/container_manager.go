package container

import (
	"bytes"
	"context"
	"io"
	"log"
	"strconv"

	"github.com/luoruofeng/dockermanagersingle/mapping"
	"github.com/luoruofeng/dockermanagersingle/types"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	dockernat "github.com/docker/go-connections/nat"

	dockerfilters "github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// var AllContainer map[string]types.ContainerInfo
// var AllImage map[string]types.ImageInfo
// var AllNetwork map[string]types.NewworkInfo
var cm ContainerManager

type ContainerManager interface {
	GetAllContainer() ([]types.ContainerInfo, error)
	GetAllImage() ([]types.ImageInfo, error)
	GetAllNetwork() ([]types.NetworkInfo, error)

	GetContainerById(id string) (dockertypes.ContainerJSON, error)
	GetImageById(id string) (dockertypes.ImageInspect, error)
	GetNetworkById(id string) (dockertypes.NetworkResource, error)

	StopContainerById(id string) error
	DeleteContainerById(id string) error
	DeleteImageById(id string) error
	DeleteNetworkById(id string) error

	PullImage(name string, version string) (io.ReadCloser, error) //需要外部关闭readerClose
	BuildImage(dockerfile string) (dockertypes.ImageBuildResponse, error)
	GetContainerLogById(id string) (io.ReadCloser, error) //需要外部关闭readerClose

	CreateContainer(imageId string, env []string, cmd []string, ports map[int]int, containerName string) (containertypes.ContainerCreateCreatedBody, error)
	StartContainer(containerID string) error

	CreateNetwork(name string, subnet string, gateway string) (dockertypes.NetworkCreateResponse, error)
	ConnectNetwork(networkId string, containerId string) error
	DisconnectNetwork(networkId string, containerId string) error
}

type DockerClient interface {
	client.ImageAPIClient
	client.ContainerAPIClient
	client.NetworkAPIClient
}

type containerManager struct {
	cli DockerClient
	ctx context.Context
}

func (cm *containerManager) CreateNetwork(name string, subnet string, gateway string) (dockertypes.NetworkCreateResponse, error) {
	var cs []networktypes.IPAMConfig
	cs = append(cs, networktypes.IPAMConfig{
		Subnet:  subnet,
		Gateway: gateway,
	})

	return cm.cli.NetworkCreate(cm.ctx, name, dockertypes.NetworkCreate{
		CheckDuplicate: true,
		IPAM: &networktypes.IPAM{
			Driver: "default",
			Config: cs,
		},
	})
}

func (cm *containerManager) ConnectNetwork(networkId string, containerId string) error {
	return cm.cli.NetworkConnect(cm.ctx, networkId, containerId, &networktypes.EndpointSettings{})
}

func (cm *containerManager) DisconnectNetwork(networkId string, containerId string) error {
	return cm.cli.NetworkDisconnect(cm.ctx, networkId, containerId, true)
}

//ports type is map[int]int k:container port v:host port
func (cm *containerManager) CreateContainer(imageId string, env []string, cmd []string, ports map[int]int, containerName string) (containertypes.ContainerCreateCreatedBody, error) {
	//set ports
	var pm map[dockernat.Port][]dockernat.PortBinding
	if len(ports) > 0 {
		pm = make(map[dockernat.Port][]dockernat.PortBinding)
		for k, v := range ports {
			cp := dockernat.Port(strconv.Itoa(v))
			hp := dockernat.PortBinding{
				HostIP:   "",
				HostPort: strconv.Itoa(k),
			}
			pbs := make([]dockernat.PortBinding, 0)
			pbs = append(pbs, hp)
			pm[cp] = pbs
		}
	}

	return cm.cli.ContainerCreate(cm.ctx,
		&container.Config{
			Image: imageId,
			Cmd:   cmd,
			Env:   env,
			Tty:   false,
		},
		&container.HostConfig{
			PortBindings: pm,
		},
		nil, nil, containerName)
}

func (cm *containerManager) StartContainer(containerID string) error {
	return cm.cli.ContainerStart(cm.ctx, containerID, dockertypes.ContainerStartOptions{})
}

func (cm *containerManager) GetContainerLogById(id string) (io.ReadCloser, error) {
	return cm.cli.ContainerLogs(cm.ctx, id, dockertypes.ContainerLogsOptions{})
}

func (cm *containerManager) BuildImage(dockerfile string) (dockertypes.ImageBuildResponse, error) {
	return cm.cli.ImageBuild(cm.ctx, bytes.NewBuffer([]byte(dockerfile)), dockertypes.ImageBuildOptions{})
}

func (cm *containerManager) PullImage(name string, version string) (io.ReadCloser, error) {
	return cm.cli.ImagePull(cm.ctx, name+":"+version, dockertypes.ImagePullOptions{})
}

func (cm *containerManager) StopContainerById(id string) error {
	err := cm.cli.ContainerStop(cm.ctx, id, nil)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (cm *containerManager) DeleteContainerById(id string) error {
	err := cm.cli.ContainerStop(cm.ctx, id, nil)
	if err != nil {
		return err
	} else {
		return cm.cli.ContainerRemove(cm.ctx, id, dockertypes.ContainerRemoveOptions{})
	}
}

func (cm *containerManager) DeleteImageById(id string) error {
	_, err := cm.cli.ImageRemove(cm.ctx, id, dockertypes.ImageRemoveOptions{})
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (cm *containerManager) DeleteNetworkById(id string) error {
	return cm.cli.NetworkRemove(cm.ctx, id)
}

func (cm *containerManager) GetAllContainer() ([]types.ContainerInfo, error) {
	cis := make([]types.ContainerInfo, 0)

	args := dockerfilters.NewArgs(
		dockerfilters.Arg("status", "running"),
		dockerfilters.Arg("status", "exited"),
		dockerfilters.Arg("status", "created"))

	containers, err := cm.cli.ContainerList(cm.ctx, dockertypes.ContainerListOptions{All: true, Filters: args})
	if err != nil {
		return cis, err
	}

	if len(containers) > 0 {
		for _, v := range containers {
			if ic, err := mapping.ContainerOTI(v); err != nil {
				log.Println(err.Error())
				continue
			} else {
				cis = append(cis, ic)
			}
		}
	}
	return cis, nil
}

func (cm *containerManager) GetAllImage() ([]types.ImageInfo, error) {
	iis := make([]types.ImageInfo, 0)
	images, err := cm.cli.ImageList(cm.ctx, dockertypes.ImageListOptions{All: true})
	if err != nil {
		return iis, err
	}

	if len(images) > 0 {
		for _, v := range images {
			if ii, err := mapping.ImageOTI(v); err != nil {
				log.Println(err.Error())
				continue
			} else {
				iis = append(iis, ii)
			}
		}
	}
	return iis, nil
}

func (cm *containerManager) GetAllNetwork() ([]types.NetworkInfo, error) {
	nis := make([]types.NetworkInfo, 0)

	args := dockerfilters.NewArgs(
		dockerfilters.Arg("driver", "bridge"))

	networks, err := cm.cli.NetworkList(cm.ctx, dockertypes.NetworkListOptions{Filters: args})
	if err != nil {
		return nis, err
	}

	if len(networks) > 0 {
		for _, v := range networks {
			if ni, err := mapping.NetworkOTI(v); err != nil {
				log.Println(err.Error())
				continue
			} else {
				nis = append(nis, ni)
			}
		}
	}
	return nis, nil
}

func (cm *containerManager) GetContainerById(id string) (dockertypes.ContainerJSON, error) {
	return cm.cli.ContainerInspect(cm.ctx, id)
}

func (cm *containerManager) GetImageById(id string) (dockertypes.ImageInspect, error) {
	ii, _, err := cm.cli.ImageInspectWithRaw(cm.ctx, id)
	return ii, err
}
func (cm *containerManager) GetNetworkById(id string) (dockertypes.NetworkResource, error) {
	return cm.cli.NetworkInspect(cm.ctx, id, dockertypes.NetworkInspectOptions{})
}

func InitContainerManager(ctx context.Context, client *client.Client) {
	cm = &containerManager{
		ctx: ctx,
		cli: client,
	}
	// 容器管理没有闭环，占时不写这部分
	// AllContainer = make(mape[string]types.ContainerInfo)
	// AllImage = make(mape[string]types.ImageInfo)
	// AllNetwork = make(mape[string]types.NewworkInfo)
	// TODO
	// logs.Println("Read all data from docker engin to RAM")
}

func GetCM() ContainerManager {
	return cm
}
