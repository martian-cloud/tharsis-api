package resource

import (
	"bufio"
	"os"
	"sort"
	"strconv"
	"strings"

	humanize "github.com/dustin/go-humanize"
)

const procNetDevFile = "/proc/net/dev"

var _ metricsCollector = (*networkCollector)(nil)

// trafficSnapshot holds byte and packet counts for a single point in time.
type trafficSnapshot struct {
	receivedBytes   uint64
	sentBytes       uint64
	receivedPackets uint64
	sentPackets     uint64
}

type networkCollector struct {
	limits            Limits
	procNetDevPath    string
	interfaceName     string
	baseline          *trafficSnapshot // counters at monitor start; nil if unavailable
	usage             trafficSnapshot  // delta from baseline, updated each poll
	readOK            bool             // distinguishes "genuinely zero" from "never read" so writeMetrics reports nil, not 0
	receivedThreshold thresholdTracker
	sentThreshold     thresholdTracker
}

func newNetworkCollector(limits Limits) *networkCollector {
	c := &networkCollector{
		limits:            limits,
		procNetDevPath:    procNetDevFile,
		receivedThreshold: newThresholdTracker(limits.NetworkReceivedBytes),
		sentThreshold:     newThresholdTracker(limits.NetworkSentBytes),
	}

	c.interfaceName = findNetworkInterface(c.procNetDevPath)
	c.baseline = c.readTrafficCounters()

	return c
}

func (c *networkCollector) finalizeSample() {
	if c.baseline == nil {
		return
	}

	current := c.readTrafficCounters()
	if current == nil {
		return
	}

	c.readOK = true

	// >= guards against counter resets on each field; skip the update rather than underflow.
	if current.receivedBytes >= c.baseline.receivedBytes {
		c.usage.receivedBytes = current.receivedBytes - c.baseline.receivedBytes
	}
	if current.sentBytes >= c.baseline.sentBytes {
		c.usage.sentBytes = current.sentBytes - c.baseline.sentBytes
	}
	if current.receivedPackets >= c.baseline.receivedPackets {
		c.usage.receivedPackets = current.receivedPackets - c.baseline.receivedPackets
	}
	if current.sentPackets >= c.baseline.sentPackets {
		c.usage.sentPackets = current.sentPackets - c.baseline.sentPackets
	}
}

func (c *networkCollector) checkWarnings(logger Logger) {
	if warn, pct := c.receivedThreshold.check(c.usage.receivedBytes); warn {
		logger.Warningf("WARNING: Job has exceeded %d%% of network received limit %s (current: %s)",
			pct, humanize.IBytes(c.limits.NetworkReceivedBytes), humanize.IBytes(c.usage.receivedBytes))
	}

	if warn, pct := c.sentThreshold.check(c.usage.sentBytes); warn {
		logger.Warningf("WARNING: Job has exceeded %d%% of network sent limit %s (current: %s)",
			pct, humanize.IBytes(c.limits.NetworkSentBytes), humanize.IBytes(c.usage.sentBytes))
	}
}

// checkBreaches reports either network direction at or over the critical percentage of its limit.
func (c *networkCollector) checkBreaches() []Breach {
	var breaches []Breach

	for _, candidate := range []struct {
		tracker *thresholdTracker
		kind    Kind
		usage   uint64
	}{
		{&c.receivedThreshold, ResourceNetworkRecv, c.usage.receivedBytes},
		{&c.sentThreshold, ResourceNetworkSent, c.usage.sentBytes},
	} {
		if b, ok := candidate.tracker.breach(candidate.kind, candidate.usage); ok {
			breaches = append(breaches, b)
		}
	}

	return breaches
}

func (c *networkCollector) writeMetrics(m *Metrics) {
	if !c.readOK {
		return
	}

	rb := int64(c.usage.receivedBytes)
	m.TotalNetworkReceivedBytes = &rb

	sb := int64(c.usage.sentBytes)
	m.TotalNetworkSentBytes = &sb

	rp := int64(c.usage.receivedPackets)
	m.TotalNetworkReceivedPackets = &rp

	sp := int64(c.usage.sentPackets)
	m.TotalNetworkSentPackets = &sp
}

// readTrafficCounters returns the current counters for c.interfaceName, or nil on error.
func (c *networkCollector) readTrafficCounters() *trafficSnapshot {
	if c.interfaceName == "" {
		return nil
	}

	counters, err := readNetDev(c.procNetDevPath)
	if err != nil {
		return nil
	}

	snapshot, ok := counters[c.interfaceName]
	if !ok {
		return nil
	}

	return &snapshot
}

// findNetworkInterface picks the primary non-loopback network interface to monitor, preferring eth0.
func findNetworkInterface(procNetDevPath string) string {
	counters, err := readNetDev(procNetDevPath)
	if err != nil {
		return ""
	}

	if _, ok := counters["eth0"]; ok {
		return "eth0"
	}

	names := make([]string, 0, len(counters))
	for name := range counters {
		if name != "lo" {
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		return ""
	}

	sort.Strings(names)

	return names[0]
}

// readNetDev parses the given /proc/net/dev file into per-interface traffic counters.
func readNetDev(path string) (map[string]trafficSnapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	counters := map[string]trafficSnapshot{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		name, rest, found := strings.Cut(scanner.Text(), ":")
		if !found {
			// The two header lines have no colon-terminated interface name.
			continue
		}

		fields := strings.Fields(rest)
		if len(fields) < 16 {
			continue
		}

		// Receive columns are 0-7, transmit columns are 8-15; bytes at 0/8, packets at 1/9.
		rxBytes, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}

		rxPackets, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		txBytes, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			continue
		}

		txPackets, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}

		counters[strings.TrimSpace(name)] = trafficSnapshot{
			receivedBytes:   rxBytes,
			sentBytes:       txBytes,
			receivedPackets: rxPackets,
			sentPackets:     txPackets,
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return counters, nil
}
