package resource

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
)

const (
	// cgroup v2 detection and path resolution.
	cgroupV2ControllersFile = "/sys/fs/cgroup/cgroup.controllers"
	cgroupSelfFile          = "/proc/self/cgroup"
	cgroupV2BaseDir         = "/sys/fs/cgroup"
	cgroupV2HierarchyID     = "0"

	// cgroup v2 stat files and field names.
	cgroupV2MemoryFile      = "memory.current"
	cgroupV2MemoryPeakFile  = "memory.peak"
	cgroupV2MemoryStatFile  = "memory.stat"
	cgroupV2InactiveFileKey = "inactive_file"
	cgroupV2CPUFile         = "cpu.stat"
	cgroupV2CPUUsecKey      = "usage_usec"
	cgroupV2IOFile          = "io.stat"
	cgroupV2IOReadKey       = "rbytes"
	cgroupV2IOWriteKey      = "wbytes"
)

var _ metricsCollector = (*cgroupCollector)(nil)

// cgroupCollector reads memory, CPU, and disk I/O from the container's own cgroup v2 files.
type cgroupCollector struct {
	limits          Limits
	memoryDir       string
	cpuDir          string
	diskDir         string
	peakMemoryBytes uint64
	workingSetBytes uint64
	lastCPUMicros   uint64
	totalCPUMicros  uint64
	// diskRead/WriteBytes are the monitored-period totals (raw counter minus baseline).
	diskReadBytes  uint64
	diskWriteBytes uint64
	// diskReadBaseline and diskWriteBaseline isolate I/O to the monitored period.
	diskReadBaseline  uint64
	diskWriteBaseline uint64
	// read-succeeded flags distinguish "genuinely zero" from "never read" so writeMetrics reports nil, not 0.
	memReadOK          bool
	cpuReadOK          bool
	diskReadOK         bool
	memThreshold       thresholdTracker
	diskReadThreshold  thresholdTracker
	diskWriteThreshold thresholdTracker
}

func newCgroupCollector(limits Limits) *cgroupCollector {
	c := &cgroupCollector{
		limits:             limits,
		memThreshold:       newThresholdTracker(limits.MemoryBytes),
		diskReadThreshold:  newThresholdTracker(limits.DiskReadBytes),
		diskWriteThreshold: newThresholdTracker(limits.DiskWriteBytes),
	}

	c.memoryDir, c.cpuDir, c.diskDir = detectCgroup()

	// Snapshot disk I/O baseline so metrics report only monitored-period I/O.
	if c.diskDir != "" {
		if r, w, err := readDiskBytes(c.diskDir + "/" + cgroupV2IOFile); err == nil {
			c.diskReadBaseline = r
			c.diskWriteBaseline = w
		}
	}

	// Prime the CPU baseline so the first tick's delta counts from monitor start, not the first sample.
	if c.cpuDir != "" {
		if usec, err := c.readCPUMicros(); err == nil {
			c.lastCPUMicros = usec
		}
	}

	return c
}

func (c *cgroupCollector) finalizeSample() {
	if c.memoryDir != "" {
		if peak, err := readUint64File(c.memoryDir + "/" + cgroupV2MemoryPeakFile); err == nil {
			// Kernel-tracked peak captures spikes between samples and excludes pre-monitoring baseline concerns.
			c.peakMemoryBytes = peak
			c.memReadOK = true
		}

		if mem, err := readUint64File(c.memoryDir + "/" + cgroupV2MemoryFile); err == nil {
			// Fallback peak-tracking when memory.peak is absent (kernels older than 5.19); a no-op when it isn't.
			if mem > c.peakMemoryBytes {
				c.peakMemoryBytes = mem
			}

			c.workingSetBytes = mem
			// Exclude reclaimable page cache, which fills toward the limit on I/O-heavy jobs without OOM risk.
			if inactive, err := readStatField(c.memoryDir+"/"+cgroupV2MemoryStatFile, cgroupV2InactiveFileKey); err == nil && inactive < mem {
				c.workingSetBytes = mem - inactive
			}

			c.memReadOK = true
		}
	}

	if c.cpuDir != "" {
		if usec, err := c.readCPUMicros(); err == nil {
			if usec >= c.lastCPUMicros {
				c.totalCPUMicros += usec - c.lastCPUMicros
			}

			c.lastCPUMicros = usec
			c.cpuReadOK = true
		}
	}

	if c.diskDir != "" {
		if r, w, err := readDiskBytes(c.diskDir + "/" + cgroupV2IOFile); err == nil {
			c.diskReadBytes = subtractBaseline(r, c.diskReadBaseline)
			c.diskWriteBytes = subtractBaseline(w, c.diskWriteBaseline)
			c.diskReadOK = true
		}
	}
}

func (c *cgroupCollector) checkWarnings(logger Logger) {
	if warn, pct := c.memThreshold.check(c.workingSetBytes); warn {
		logger.Warningf("WARNING: Job has exceeded %d%% of %s memory limit (current: %s)",
			pct, humanize.IBytes(c.limits.MemoryBytes), humanize.IBytes(c.workingSetBytes))
		// Ship immediately since an OOM kill of the whole container would drop the buffered warning.
		logger.Flush()
	}

	if warn, pct := c.diskReadThreshold.check(c.diskReadBytes); warn {
		logger.Warningf("WARNING: Job has exceeded %d%% of disk read limit %s (current: %s)",
			pct, humanize.IBytes(c.limits.DiskReadBytes), humanize.IBytes(c.diskReadBytes))
	}

	if warn, pct := c.diskWriteThreshold.check(c.diskWriteBytes); warn {
		logger.Warningf("WARNING: Job has exceeded %d%% of disk write limit %s (current: %s)",
			pct, humanize.IBytes(c.limits.DiskWriteBytes), humanize.IBytes(c.diskWriteBytes))
	}
}

// checkBreaches reports any cgroup-tracked resource at or over the critical percentage of its limit.
func (c *cgroupCollector) checkBreaches() []Breach {
	var breaches []Breach

	for _, candidate := range []struct {
		tracker *thresholdTracker
		kind    Kind
		usage   uint64
	}{
		{&c.memThreshold, ResourceMemory, c.workingSetBytes},
		{&c.diskReadThreshold, ResourceDiskRead, c.diskReadBytes},
		{&c.diskWriteThreshold, ResourceDiskWrite, c.diskWriteBytes},
	} {
		if b, ok := candidate.tracker.breach(candidate.kind, candidate.usage); ok {
			breaches = append(breaches, b)
		}
	}

	return breaches
}

func (c *cgroupCollector) writeMetrics(m *Metrics) {
	if c.memReadOK {
		peak := int64(c.peakMemoryBytes)
		m.PeakMemoryBytes = &peak
	}

	if c.cpuReadOK {
		cpuMS := float64(c.totalCPUMicros) / 1000
		m.TotalCPUTimeMS = &cpuMS
	}

	if c.diskReadOK {
		dr := int64(c.diskReadBytes)
		dw := int64(c.diskWriteBytes)
		m.TotalDiskReadBytes = &dr
		m.TotalDiskWriteBytes = &dw
	}
}

// readCPUMicros reads cumulative CPU time in microseconds from cpu.stat.
func (c *cgroupCollector) readCPUMicros() (uint64, error) {
	return readStatField(c.cpuDir+"/"+cgroupV2CPUFile, cgroupV2CPUUsecKey)
}

// detectCgroup resolves the cgroup v2 memory, CPU, and disk directories.
func detectCgroup() (memoryDir, cpuDir, diskDir string) {
	// The unified-hierarchy controllers file exists only on cgroup v2.
	if _, err := os.Stat(cgroupV2ControllersFile); err != nil {
		return "", "", ""
	}

	path, err := findCgroupPath(cgroupSelfFile, cgroupV2HierarchyID)
	if err != nil {
		return "", "", ""
	}

	memoryDir = resolveCgroupDir(cgroupV2BaseDir, path, cgroupV2MemoryFile)
	cpuDir = resolveCgroupDir(cgroupV2BaseDir, path, cgroupV2CPUFile)
	diskDir = resolveCgroupDir(cgroupV2BaseDir, path, cgroupV2IOFile)

	return memoryDir, cpuDir, diskDir
}

// resolveCgroupDir joins baseDir and the cgroup-relative path, falling back to baseDir where the container's own cgroup is bind-mounted.
func resolveCgroupDir(baseDir, path, statFile string) string {
	dir := baseDir + path
	if _, err := os.Stat(dir + "/" + statFile); err == nil {
		return dir
	}

	if _, err := os.Stat(baseDir + "/" + statFile); err == nil {
		return baseDir
	}

	return ""
}

// findCgroupPath parses the given /proc/self/cgroup file and returns the path for the given subsystem.
func findCgroupPath(cgroupFile, subsystem string) (string, error) {
	f, err := os.Open(cgroupFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 3)
		if len(parts) != 3 {
			continue
		}
		if parts[0] == subsystem || parts[1] == subsystem {
			return parts[2], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("cgroup subsystem %q not found in %s", subsystem, cgroupFile)
}

// readUint64File reads a file containing a single uint64 value.
func readUint64File(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

// readStatField scans a key-value stat file (e.g. cpu.stat) for a named field.
func readStatField(path, key string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 2 && parts[0] == key {
			return strconv.ParseUint(parts[1], 10, 64)
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return 0, fmt.Errorf("key %q not found in %s", key, path)
}

// readDiskBytes parses io.stat (cgroup v2) and sums read/write bytes across all devices.
func readDiskBytes(path string) (readBytes, writeBytes uint64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		for _, field := range fields[1:] {
			kv := strings.SplitN(field, "=", 2)
			if len(kv) != 2 {
				continue
			}

			v, pErr := strconv.ParseUint(kv[1], 10, 64)
			if pErr != nil {
				continue
			}

			switch kv[0] {
			case cgroupV2IOReadKey:
				readBytes += v
			case cgroupV2IOWriteKey:
				writeBytes += v
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}

	return readBytes, writeBytes, nil
}

// subtractBaseline returns current minus baseline, guarding against counter resets with >=.
func subtractBaseline(current, baseline uint64) uint64 {
	if current < baseline {
		return 0
	}

	return current - baseline
}
