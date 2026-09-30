package resource

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadNetDev(t *testing.T) {
	const content = `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000      10    0    0    0     0          0         0     1000      10    0    0    0     0       0          0
  eth0: 5000      50    0    0    0     0          0         0     3000      30    0    0    0     0       0          0
  eth1: 7000      70    1    2    0     0          0         0     4000      40    0    0    0     0       0          0
`

	counters, err := readNetDev(writeTempFile(t, content))
	require.NoError(t, err)

	assert.Len(t, counters, 3)

	assert.Equal(t, trafficSnapshot{
		receivedBytes:   5000,
		sentBytes:       3000,
		receivedPackets: 50,
		sentPackets:     30,
	}, counters["eth0"])

	assert.Equal(t, trafficSnapshot{
		receivedBytes:   1000,
		sentBytes:       1000,
		receivedPackets: 10,
		sentPackets:     10,
	}, counters["lo"])
}

func TestReadNetDevMissingFile(t *testing.T) {
	_, err := readNetDev(filepath.Join(t.TempDir(), "missing"))
	assert.Error(t, err)
}

func TestNetworkCollectorWriteMetrics(t *testing.T) {
	t.Run("baseline without a successful sample emits nothing", func(t *testing.T) {
		c := &networkCollector{baseline: &trafficSnapshot{}}

		m := &Metrics{}
		c.writeMetrics(m)

		assert.Nil(t, m.TotalNetworkReceivedBytes)
		assert.Nil(t, m.TotalNetworkSentBytes)
	})

	t.Run("emits totals", func(t *testing.T) {
		c := &networkCollector{
			readOK: true,
			usage: trafficSnapshot{
				receivedBytes:   1000,
				sentBytes:       2000,
				receivedPackets: 10,
				sentPackets:     20,
			},
		}

		m := &Metrics{}
		c.writeMetrics(m)

		require.NotNil(t, m.TotalNetworkReceivedBytes)
		assert.Equal(t, int64(1000), *m.TotalNetworkReceivedBytes)
		require.NotNil(t, m.TotalNetworkSentBytes)
		assert.Equal(t, int64(2000), *m.TotalNetworkSentBytes)
		require.NotNil(t, m.TotalNetworkReceivedPackets)
		assert.Equal(t, int64(10), *m.TotalNetworkReceivedPackets)
		require.NotNil(t, m.TotalNetworkSentPackets)
		assert.Equal(t, int64(20), *m.TotalNetworkSentPackets)
	})

	t.Run("zero usage writes zero, not nil, when a sample was read", func(t *testing.T) {
		c := &networkCollector{readOK: true}

		m := &Metrics{}
		c.writeMetrics(m)

		// A collected-but-idle interface is a real zero measurement, distinct from "never collected".
		require.NotNil(t, m.TotalNetworkReceivedBytes)
		assert.Equal(t, int64(0), *m.TotalNetworkReceivedBytes)
		require.NotNil(t, m.TotalNetworkSentBytes)
		assert.Equal(t, int64(0), *m.TotalNetworkSentBytes)
		require.NotNil(t, m.TotalNetworkReceivedPackets)
		assert.Equal(t, int64(0), *m.TotalNetworkReceivedPackets)
		require.NotNil(t, m.TotalNetworkSentPackets)
		assert.Equal(t, int64(0), *m.TotalNetworkSentPackets)
	})
}

func TestFindNetworkInterface(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "prefers eth0",
			content: "h\nh\n    lo: 1 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0\n" +
				"  eth1: 1 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0\n" +
				"  eth0: 1 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0\n",
			want: "eth0",
		},
		{
			name: "falls back to first non-loopback sorted",
			content: "h\nh\n    lo: 1 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0\n" +
				"  ethb: 1 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0\n" +
				"  etha: 1 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0\n",
			want: "etha",
		},
		{
			name:    "only loopback yields empty",
			content: "h\nh\n    lo: 1 0 0 0 0 0 0 0 1 0 0 0 0 0 0 0\n",
			want:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, findNetworkInterface(writeTempFile(t, test.content)))
		})
	}
}

func TestNetworkCollectorFinalizeSample(t *testing.T) {
	const header = "h\nh\n"
	row := func(rx, sent uint64) string {
		return "  eth0: " + strconv.FormatUint(rx, 10) + " 10 0 0 0 0 0 0 " + strconv.FormatUint(sent, 10) + " 20 0 0 0 0 0 0\n"
	}

	path := writeTempFile(t, header+row(1000, 2000))
	c := &networkCollector{procNetDevPath: path, interfaceName: "eth0"}
	c.baseline = c.readTrafficCounters()
	require.NotNil(t, c.baseline)

	require.NoError(t, os.WriteFile(path, []byte(header+row(6000, 8000)), 0o600))
	c.finalizeSample()

	m := &Metrics{}
	c.writeMetrics(m)

	require.NotNil(t, m.TotalNetworkReceivedBytes)
	assert.Equal(t, int64(5000), *m.TotalNetworkReceivedBytes)
	require.NotNil(t, m.TotalNetworkSentBytes)
	assert.Equal(t, int64(6000), *m.TotalNetworkSentBytes)
}

func TestNetworkCollectorCheckWarnings(t *testing.T) {
	t.Run("received warning fires from received usage", func(t *testing.T) {
		c := &networkCollector{
			limits:            Limits{NetworkReceivedBytes: 100},
			receivedThreshold: newThresholdTracker(100),
			sentThreshold:     newThresholdTracker(0),
			usage:             trafficSnapshot{receivedBytes: 85},
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 1, logger.warnings)
	})

	t.Run("sent warning fires from sent usage", func(t *testing.T) {
		c := &networkCollector{
			limits:            Limits{NetworkSentBytes: 100},
			receivedThreshold: newThresholdTracker(0),
			sentThreshold:     newThresholdTracker(100),
			usage:             trafficSnapshot{sentBytes: 85},
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 1, logger.warnings)
	})

	t.Run("received usage does not trip the sent limit", func(t *testing.T) {
		// Guards against the two thresholds being wired to the wrong usage field.
		c := &networkCollector{
			limits:            Limits{NetworkSentBytes: 100},
			receivedThreshold: newThresholdTracker(0),
			sentThreshold:     newThresholdTracker(100),
			usage:             trafficSnapshot{receivedBytes: 1000, sentBytes: 10},
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 0, logger.warnings)
	})

	t.Run("no warnings when limits are unset", func(t *testing.T) {
		c := &networkCollector{
			receivedThreshold: newThresholdTracker(0),
			sentThreshold:     newThresholdTracker(0),
			usage:             trafficSnapshot{receivedBytes: 1 << 30, sentBytes: 1 << 30},
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 0, logger.warnings)
	})
}

func TestNetworkCollectorCheckBreaches(t *testing.T) {
	t.Run("received breach reports the received kind from received usage", func(t *testing.T) {
		c := &networkCollector{
			receivedThreshold: newThresholdTracker(100),
			usage:             trafficSnapshot{receivedBytes: 96},
		}
		breaches := c.checkBreaches()
		require.Len(t, breaches, 1)
		assert.Equal(t, ResourceNetworkRecv, breaches[0].Kind)
		assert.Equal(t, uint64(96), breaches[0].Usage)
	})

	t.Run("received and sent breach independently from their own usage", func(t *testing.T) {
		c := &networkCollector{
			receivedThreshold: newThresholdTracker(100),
			sentThreshold:     newThresholdTracker(100),
			usage:             trafficSnapshot{receivedBytes: 10, sentBytes: 99}, // received below critical
		}
		breaches := c.checkBreaches()
		require.Len(t, breaches, 1)
		assert.Equal(t, ResourceNetworkSent, breaches[0].Kind)
	})

	t.Run("no breach below critical", func(t *testing.T) {
		c := &networkCollector{
			receivedThreshold: newThresholdTracker(100),
			usage:             trafficSnapshot{receivedBytes: 94},
		}
		assert.Empty(t, c.checkBreaches())
	})

	t.Run("no breach when limits unset", func(t *testing.T) {
		c := &networkCollector{usage: trafficSnapshot{receivedBytes: 1 << 40, sentBytes: 1 << 40}}
		assert.Empty(t, c.checkBreaches())
	})
}
