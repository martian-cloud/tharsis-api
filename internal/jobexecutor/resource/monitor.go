// Package resource tracks CPU, memory, network, and disk usage for a containerized job.
package resource

import (
	"os"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
)

// monitorInterval is the polling interval; cgroup and network counters are cumulative, so 1s loses no accuracy.
const monitorInterval = 1 * time.Second

// metricsCollector is implemented by each metric source that gathers data across polling cycles.
type metricsCollector interface {
	finalizeSample()             // read the latest counters into the collector's running totals
	checkWarnings(logger Logger) // emit threshold warnings to the job log
	checkBreaches() []Breach     // report any resource at/over the critical percentage of its limit
	writeMetrics(m *Metrics)     // write the final collected values into the result
}

type monitor struct {
	resultCh    chan *Metrics
	breachCh    chan Breach
	doneChannel chan struct{}
	logger      Logger
	collectors  []metricsCollector
}

// Option configures a Monitor.
type Option func(*monitorConfig)

type monitorConfig struct {
	limits *Limits
}

// WithLimits sets the resource limits used for threshold warnings.
func WithLimits(limits Limits) Option {
	return func(c *monitorConfig) {
		c.limits = &limits
	}
}

// NewMonitor creates a Monitor with the given options.
func NewMonitor(logger Logger, opts ...Option) (Monitor, error) {
	cfg := &monitorConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var limits Limits
	if cfg.limits != nil {
		limits = *cfg.limits
	} else {
		logger.Infof("resource monitor: no limits configured, monitoring usage without threshold warnings")
	}

	// Metrics are only meaningful inside a container, where cgroup and interface counters are job-scoped.
	var collectors []metricsCollector
	if isContainerized() {
		collectors = []metricsCollector{
			newCgroupCollector(limits),
			newNetworkCollector(limits),
		}
	}

	return &monitor{
		resultCh:    make(chan *Metrics, 1),
		breachCh:    make(chan Breach, 1),
		doneChannel: make(chan struct{}),
		logger:      logger,
		collectors:  collectors,
	}, nil
}

// Start begins monitoring in a background goroutine.
func (rm *monitor) Start() {
	go rm.monitor()
}

// Breaches returns the channel on which the first resource breach is delivered.
func (rm *monitor) Breaches() <-chan Breach {
	return rm.breachCh
}

// Stop signals the monitor to finish and returns the final snapshot, or nil if nothing was collected.
func (rm *monitor) Stop() *Metrics {
	close(rm.doneChannel)

	return <-rm.resultCh
}

func (rm *monitor) monitor() {
	defer func() {
		if r := recover(); r != nil {
			rm.logger.Errorf("resource monitor panicked: %v", r)
			rm.resultCh <- nil
		}
	}()

	if len(rm.collectors) == 0 {
		<-rm.doneChannel
		rm.resultCh <- nil

		return
	}

	ticker := time.NewTicker(monitorInterval)
	defer ticker.Stop()

	// breachSent ensures the executor is signaled at most once.
	breachSent := false

	for {
		select {
		case <-rm.doneChannel:
			// Take one final sample so jobs shorter than monitorInterval still report data and late OOM kills are logged.
			for _, c := range rm.collectors {
				c.finalizeSample()
				c.checkWarnings(rm.logger)
			}

			rm.resultCh <- rm.snapshot()

			return
		case <-ticker.C:
		}

		for _, c := range rm.collectors {
			c.finalizeSample()
		}

		for _, c := range rm.collectors {
			c.checkWarnings(rm.logger)
		}

		if !breachSent {
			breachSent = rm.reportBreaches()
		}
	}
}

// reportBreaches sends the first resource at/over its critical limit and reports whether one was sent; the buffered channel never blocks.
func (rm *monitor) reportBreaches() bool {
	for _, c := range rm.collectors {
		breaches := c.checkBreaches()
		if len(breaches) == 0 {
			continue
		}

		select {
		case rm.breachCh <- breaches[0]:
		default:
		}

		return true
	}

	return false
}

// snapshot writes the current collected values from every collector into a fresh Metrics.
func (rm *monitor) snapshot() *Metrics {
	m := &Metrics{}
	for _, c := range rm.collectors {
		c.writeMetrics(m)
	}

	return m
}

// LoadLimitsFromEnv reads byte-valued resource limits from the environment, ignoring unparseable values so a misconfigured limit never blocks the job.
func LoadLimitsFromEnv(lookup func(string) string, logger Logger) Limits {
	limits := Limits{}

	fields := []struct {
		env string
		dst *uint64
	}{
		{"MEMORY_LIMIT", &limits.MemoryBytes},
		{"DISK_READ_LIMIT", &limits.DiskReadBytes},
		{"DISK_WRITE_LIMIT", &limits.DiskWriteBytes},
		{"NETWORK_RECEIVED_LIMIT", &limits.NetworkReceivedBytes},
		{"NETWORK_SENT_LIMIT", &limits.NetworkSentBytes},
	}

	for _, f := range fields {
		v := lookup(f.env)
		if v == "" {
			continue
		}

		n, err := humanize.ParseBytes(v)
		if err != nil {
			logger.Warningf("resource monitor: ignoring invalid %s %q: %v", f.env, v, err)
			continue
		}

		*f.dst = n
	}

	return limits
}

func isContainerized() bool {
	// Docker creates this file in every container it starts.
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Kubernetes injects this into every pod via the default service account.
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	// ECS/Fargate injects a task metadata endpoint but creates none of the signals above.
	if os.Getenv("ECS_CONTAINER_METADATA_URI_V4") != "" || os.Getenv("ECS_CONTAINER_METADATA_URI") != "" {
		return true
	}

	// Fallback for OCI runtimes (containerd, CRI-O) that don't create /.dockerenv.
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		s := string(data)
		return strings.Contains(s, "kubepods") ||
			strings.Contains(s, "docker") ||
			strings.Contains(s, "containerd") ||
			strings.Contains(s, "/ecs/")
	}

	return false
}
