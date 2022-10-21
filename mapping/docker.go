package mapping

import (
	"github.com/luoruofeng/dockermanagersingle/types"

	dockertypes "github.com/docker/docker/api/types"
)

// outter's container(https://pkg.go.dev/github.com/docker/docker@v20.10.20+incompatible/api/types#Container) to innter' types.ContainerInfo
func ContainerOTI(o dockertypes.Container) (ci types.ContainerInfo, err error) {
	ci.ID = o.ID
	ci.Names = o.Names
	ci.Image = o.Image
	ci.ImageID = o.ImageID
	ci.Command = o.Command
	ci.Created = o.Created
	ci.Ports = o.Ports
	ci.SizeRw = o.SizeRw
	ci.SizeRootFs = o.SizeRootFs
	ci.Labels = o.Labels
	ci.State = o.State
	ci.Status = o.Status
	ci.HostConfig = o.HostConfig
	ci.NetworkIds = make([]string, 0)
	if o.NetworkSettings != nil && len(o.NetworkSettings.Networks) > 0 {
		for _, v := range o.NetworkSettings.Networks {
			ci.NetworkIds = append(ci.NetworkIds, v.NetworkID)
		}
	}
	return ci, err
}

// outter's container(https://pkg.go.dev/github.com/docker/docker@v20.10.20+incompatible/api/types#ImageSummary) to innter' types.ContainerInfo
func ImageOTI(o dockertypes.ImageSummary) (ii types.ImageInfo, err error) {
	ii = types.ImageInfo(o)
	return ii, err
}

// outter's container(https://pkg.go.dev/github.com/docker/docker@v20.10.20+incompatible/api/types#ImageSummary) to innter' types.ContainerInfo
func NetworkOTI(o dockertypes.NetworkResource) (ni types.NetworkInfo, err error) {
	ni = types.NetworkInfo(o)
	return ni, err
}
