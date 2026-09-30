package resource

// Kind identifies which resource limit was breached.
type Kind string

// Kind values for the resources that carry a configurable limit.
const (
	ResourceMemory      Kind = "memory"
	ResourceDiskRead    Kind = "disk read"
	ResourceDiskWrite   Kind = "disk write"
	ResourceNetworkRecv Kind = "network received"
	ResourceNetworkSent Kind = "network sent"
)

// Breach reports that a resource reached the critical percentage of its limit.
type Breach struct {
	Kind    Kind
	Limit   uint64
	Usage   uint64
	Percent int
}

// Metrics holds resource usage; a nil field means the metric was not collected (e.g. non-Linux host).
type Metrics struct {
	TotalCPUTimeMS              *float64 `json:"totalCpuTimeMs,omitempty"`
	TotalNetworkReceivedBytes   *int64   `json:"totalNetworkReceivedBytes,omitempty"`
	TotalNetworkSentBytes       *int64   `json:"totalNetworkSentBytes,omitempty"`
	TotalNetworkReceivedPackets *int64   `json:"totalNetworkReceivedPackets,omitempty"`
	TotalNetworkSentPackets     *int64   `json:"totalNetworkSentPackets,omitempty"`
	TotalDiskReadBytes          *int64   `json:"totalDiskReadBytes,omitempty"`
	TotalDiskWriteBytes         *int64   `json:"totalDiskWriteBytes,omitempty"`
	PeakMemoryBytes             *int64   `json:"peakMemoryBytes,omitempty"`
}

// Limits holds configurable resource limits used for threshold warnings; a zero field disables that warning.
type Limits struct {
	MemoryBytes          uint64
	NetworkReceivedBytes uint64
	NetworkSentBytes     uint64
	DiskReadBytes        uint64
	DiskWriteBytes       uint64
}

// Logger is the minimal logging interface required by the resource monitor.
type Logger interface {
	Infof(format string, args ...any)
	Warningf(format string, args ...any)
	Errorf(format string, args ...any)
	Flush()
}

// Monitor tracks CPU, memory, network, and disk usage for the job process tree.
type Monitor interface {
	Start()
	Breaches() <-chan Breach
	Stop() *Metrics
}
