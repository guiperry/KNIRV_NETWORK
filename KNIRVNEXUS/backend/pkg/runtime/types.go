// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package runtime

import (
	"time"

	"github.com/knirvcorp/knirvbase/go"
)

// RuntimeMode defines the execution mode for a container
type RuntimeMode string

const (
	RuntimeModeDVE    RuntimeMode = "dve"       // Standard validation environment
	RuntimeModeTEE    RuntimeMode = "tee"       // Hardware TEE (SGX/SEV/TDX)
	RuntimeModeKata   RuntimeMode = "kata"      // Kata containers (VM-based)
	RuntimeModeDocker RuntimeMode = "docker"    // Docker containers
	RuntimeModeNative RuntimeMode = "native-go" // Native Go isolation (Kali)
	RuntimeModeObject RuntimeMode = "object"    // NOC mode (universal)
)

// ObjectType defines the type of servable object
type ObjectType string

const (
	ObjectTypeWebApp     ObjectType = "webapp"     // Web application
	ObjectTypeAPI        ObjectType = "api"        // API service
	ObjectTypeBlockchain ObjectType = "blockchain" // Blockchain node
	ObjectTypeOracle     ObjectType = "oracle"     // Oracle daemon
	ObjectTypeModel      ObjectType = "model"      // Model server
	ObjectType3D         ObjectType = "3d"         // 3D GLB/GLTF object
	ObjectTypeP2P        ObjectType = "p2p"        // P2P router/node
	ObjectTypeFile       ObjectType = "file"       // File server
	ObjectTypeCustom     ObjectType = "custom"     // Custom service
)

// SecurityLevel defines the isolation strength
type SecurityLevel string

const (
	SecurityLevelBasic   SecurityLevel = "basic"   // Namespace isolation
	SecurityLevelStrong  SecurityLevel = "strong"  // + seccomp + AppArmor/SELinux
	SecurityLevelExtreme SecurityLevel = "extreme" // + hardware TEE
)

// ContainerStatus represents container lifecycle status
type ContainerStatus string

const (
	ContainerStatusCreating ContainerStatus = "creating"
	ContainerStatusRunning  ContainerStatus = "running"
	ContainerStatusStopped  ContainerStatus = "stopped"
	ContainerStatusFailed   ContainerStatus = "failed"
)

// UnifiedContainer represents a single execution unit with DVE/TEE/Container capabilities
type UnifiedContainer struct {
	ID            string          `json:"id"`
	Mode          RuntimeMode     `json:"mode"`
	ObjectType    ObjectType      `json:"object_type"`
	SecurityLevel SecurityLevel   `json:"security_level"`
	Spec          *ContainerSpec  `json:"spec"`
	Status        ContainerStatus `json:"status"`
	CreatedAt     time.Time       `json:"created_at"`

	// DVE capabilities
	ValidationRole string   `json:"validation_role,omitempty"`
	P2PEndpoints   []string `json:"p2p_endpoints,omitempty"`

	// TEE capabilities
	TEEAttester *teesecurity.TEEAttester `json:"tee_attester,omitempty"`
	Attestation *teesecurity.Attestation `json:"attestation,omitempty"`

	// Container runtime
	Runtime   ContainerRuntime `json:"runtime"`
	PID       int              `json:"pid,omitempty"`
	Namespace *LinuxNamespace  `json:"namespace,omitempty"`

	// eBPF security
	EBPFMonitor *ebpf.SyscallMonitor `json:"-"`
	SandboxLSM  *ebpf.SandboxLSM     `json:"-"`

	// NOC specific
	ObjectConfig  *NestedObjectConfig `json:"object_config,omitempty"`
	ViewportProxy ViewportProxy       `json:"-"`
	CryptoHash    string              `json:"crypto_hash,omitempty"`

	// 3D Asset support
	GLBRenderer   GLBRenderer    `json:"-"`
	AssetMetadata *AssetMetadata `json:"asset_metadata,omitempty"`

	// Model server integration (for model-type objects)
	ModelServer EmbeddedModelServer `json:"-"`
}

// ContainerSpec defines the container specification
type ContainerSpec struct {
	ID            string            `json:"id"`
	Image         string            `json:"image"`
	Command       []string          `json:"command,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	Volumes       []VolumeMount     `json:"volumes,omitempty"`
	Ports         []PortMapping     `json:"ports,omitempty"`
	Resources     ResourceLimits    `json:"resources"`
	NetworkMode   string            `json:"network_mode"`
	CapabilitySet []string          `json:"capabilities,omitempty"`
}

// VolumeMount defines a volume mount
type VolumeMount struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only"`
}

// PortMapping defines a port mapping
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"` // tcp, udp
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	CPUCores int   `json:"cpu_cores"`
	MemoryMB int64 `json:"memory_mb"`
	DiskMB   int64 `json:"disk_mb"`
}

// NestedObjectConfig defines NOC-specific configuration
type NestedObjectConfig struct {
	ObjectType        ObjectType             `json:"object_type"`
	EnableViewport    bool                   `json:"enable_viewport"`
	ViewportRenderers []string               `json:"viewport_renderers"` // http, webrtc, webgl, vnc
	ServicePorts      map[string]int         `json:"service_ports"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// AssetMetadata defines metadata for 3D assets
type AssetMetadata struct {
	Format      string         `json:"format"` // glb, gltf, obj, etc.
	FileSize    int64          `json:"file_size"`
	FilePath    string         `json:"file_path"`
	Dimensions  *Dimensions3D  `json:"dimensions"`
	Materials   []string       `json:"materials"`
	Animations  []string       `json:"animations"`
	Textures    []TextureInfo  `json:"textures"`
	Polycount   int            `json:"polycount"`
	BoundingBox *BoundingBox3D `json:"bounding_box"`
}

// Dimensions3D represents 3D object dimensions
type Dimensions3D struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Depth  float64 `json:"depth"`
}

// BoundingBox3D represents a 3D bounding box
type BoundingBox3D struct {
	Min Vector3D `json:"min"`
	Max Vector3D `json:"max"`
}

// Vector3D represents a 3D vector
type Vector3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// TextureInfo represents texture metadata
type TextureInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Resolution string `json:"resolution"`
	Format     string `json:"format"`
}

// ContainerRuntime represents the runtime interface
type ContainerRuntime interface {
	Create(spec *ContainerSpec) error
	Start() error
	Stop() error
	Delete() error
	GetStatus() ContainerStatus
}

// LinuxNamespace represents Linux namespace configuration
type LinuxNamespace struct {
	PID     int    `json:"pid"`
	Net     int    `json:"net"`
	Mount   int    `json:"mount"`
	IPC     int    `json:"ipc"`
	UTS     int    `json:"uts"`
	User    int    `json:"user"`
	Cgroup  int    `json:"cgroup"`
	RootDir string `json:"root_dir"`
}

// ViewportProxy interface for viewport management
type ViewportProxy interface {
	Start() error
	Stop() error
	HandleHTTP(path string) ([]byte, error)
	GetStatus() string
}

// GLBRenderer interface for 3D rendering
type GLBRenderer interface {
	LoadAsset(path string) error
	RenderFrame() ([]byte, error)
	GetMetadata() *AssetMetadata
	Close() error
}

// EmbeddedModelServer interface for model serving
type EmbeddedModelServer interface {
	LoadModel(path string) error
	Inference(input []byte) ([]byte, error)
	GetModelInfo() map[string]interface{}
	Close() error
}
