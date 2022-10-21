package container

import (
	"context"
	"log"

	"github.com/luoruofeng/dockermanagersingle/mapping"
	"github.com/luoruofeng/dockermanagersingle/types"

	dockertypes "github.com/docker/docker/api/types"

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
