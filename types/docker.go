package types

import (
	"time"

	dockertypes "github.com/docker/docker/api/types"
	dockernetwork "github.com/docker/docker/api/types/network"
)

type ContainerInfo struct {
	ID         string             `json:"Id"`
	Names      []string           `json:"names"`
	Image      string             `json:"image"`
	ImageID    string             `json:"image_id"`
	Command    string             `json:"command"`
	Created    int64              `json:"created"`
	Ports      []dockertypes.Port `json:"ports"`
	SizeRw     int64              `json:"size_rw,omitempty"`
	SizeRootFs int64              `json:",size_root_fs,omitempty"`
	Labels     map[string]string  `json:"labels"`
	State      string             `json:"state"`
	Status     string             `json:"status"`
	HostConfig struct {
		NetworkMode string `json:",omitempty"`
	} `json:"host_config,omitempty"`
	NetworkIds []string `json:"network_ids"`

	//https://pkg.go.dev/github.com/docker/docker@v20.10.20+incompatible/api/types#Container
	// NetworkSettings *SummaryNetworkSettings
	// Mounts          []MountPoint
}

type ImageInfo struct {
	// containers
	// Required: true
	Containers int64 `json:"Containers"`

	// created
	// Required: true
	Created int64 `json:"Created"`

	// Id
	// Required: true
	ID string `json:"Id"`

	// labels
	// Required: true
	Labels map[string]string `json:"Labels"`

	// parent Id
	// Required: true
	ParentID string `json:"ParentId"`

	// repo digests
	// Required: true
	RepoDigests []string `json:"RepoDigests"`

	// repo tags
	// Required: true
	RepoTags []string `json:"RepoTags"`

	// shared size
	// Required: true
	SharedSize int64 `json:"SharedSize"`

	// size
	// Required: true
	Size int64 `json:"Size"`

	// virtual size
	// Required: true
	VirtualSize int64 `json:"VirtualSize"`
}

type NetworkInfo struct {
	Name       string                                  // Name is the requested name of the network
	ID         string                                  `json:"Id"` // ID uniquely identifies a network on a single machine
	Created    time.Time                               // Created is the time the network created
	Scope      string                                  // Scope describes the level at which the network exists (e.g. `swarm` for cluster-wide or `local` for machine level)
	Driver     string                                  // Driver is the Driver name used to create the network (e.g. `bridge`, `overlay`)
	EnableIPv6 bool                                    // EnableIPv6 represents whether to enable IPv6
	IPAM       dockernetwork.IPAM                      // IPAM is the network's IP Address Management
	Internal   bool                                    // Internal represents if the network is used internal only
	Attachable bool                                    // Attachable represents if the global scope is manually attachable by regular containers from workers in swarm mode.
	Ingress    bool                                    // Ingress indicates the network is providing the routing-mesh for the swarm cluster.
	ConfigFrom dockernetwork.ConfigReference           // ConfigFrom specifies the source which will provide the configuration for this network.
	ConfigOnly bool                                    // ConfigOnly networks are place-holder networks for network configurations to be used by other networks. ConfigOnly networks cannot be used directly to run containers or services.
	Containers map[string]dockertypes.EndpointResource // Containers contains endpoints belonging to the network
	Options    map[string]string                       // Options holds the network specific options to use for when creating the network
	Labels     map[string]string                       // Labels holds metadata specific to the network being created
	Peers      []dockernetwork.PeerInfo                `json:",omitempty"` // List of peer nodes for an overlay network
	Services   map[string]dockernetwork.ServiceInfo    `json:",omitempty"`
}

type ImageLog struct{}
type ContainerLog struct{}
