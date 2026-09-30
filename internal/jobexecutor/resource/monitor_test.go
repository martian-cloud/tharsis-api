package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLogger struct {
	warnings int
	errors   int
	flushes  int
}

func (l *stubLogger) Infof(string, ...interface{})    {}
func (l *stubLogger) Warningf(string, ...interface{}) { l.warnings++ }
func (l *stubLogger) Errorf(string, ...interface{})   { l.errors++ }
func (l *stubLogger) Flush()                          { l.flushes++ }

func TestNewMonitor(t *testing.T) {
	logger := &stubLogger{}

	t.Run("valid with explicit limits", func(t *testing.T) {
		m, err := NewMonitor(logger, WithLimits(Limits{MemoryBytes: 1}))
		require.NoError(t, err)
		assert.NotNil(t, m)
	})

	t.Run("valid with no options", func(t *testing.T) {
		m, err := NewMonitor(logger)
		require.NoError(t, err)
		assert.NotNil(t, m)
	})
}

func TestMonitorStopWithoutCollectors(t *testing.T) {
	// Built directly so the result doesn't depend on whether the test host is containerized, which it is in CI.
	m := &monitor{
		resultCh:    make(chan *Metrics, 1),
		doneChannel: make(chan struct{}),
		logger:      &stubLogger{},
	}

	m.Start()
	assert.Nil(t, m.Stop())
}

// fakeCollector records how many times it was sampled and emits a sentinel metric.
type fakeCollector struct {
	samples  int
	breaches []Breach
}

func (f *fakeCollector) finalizeSample()         { f.samples++ }
func (f *fakeCollector) checkWarnings(Logger)    {}
func (f *fakeCollector) checkBreaches() []Breach { return f.breaches }

func (f *fakeCollector) writeMetrics(m *Metrics) {
	v := int64(f.samples)
	m.PeakMemoryBytes = &v
}

func TestMonitorFinalSampleBeforeSnapshot(t *testing.T) {
	// A job that stops before the first tick must still get one sample and report data.
	fake := &fakeCollector{}
	m := &monitor{
		resultCh:    make(chan *Metrics, 1),
		doneChannel: make(chan struct{}),
		logger:      &stubLogger{},
		collectors:  []metricsCollector{fake},
	}

	m.Start()
	metrics := m.Stop()

	require.NotNil(t, metrics)
	require.NotNil(t, metrics.PeakMemoryBytes)
	// The final sample in the done branch ran at least once despite no tick elapsing.
	assert.GreaterOrEqual(t, *metrics.PeakMemoryBytes, int64(1))
}

func TestLoadLimitsFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		env          map[string]string
		want         Limits
		wantWarnings int
	}{
		{
			name: "all limits parsed",
			env: map[string]string{
				"MEMORY_LIMIT":           "1GB",
				"DISK_READ_LIMIT":        "500MB",
				"DISK_WRITE_LIMIT":       "500MB",
				"NETWORK_RECEIVED_LIMIT": "2GB",
				"NETWORK_SENT_LIMIT":     "2GB",
			},
			want: Limits{
				MemoryBytes:          1000000000,
				DiskReadBytes:        500000000,
				DiskWriteBytes:       500000000,
				NetworkReceivedBytes: 2000000000,
				NetworkSentBytes:     2000000000,
			},
		},
		{
			name: "unset limits stay zero",
			env: map[string]string{
				"MEMORY_LIMIT": "512MB",
			},
			want: Limits{
				MemoryBytes: 512000000,
			},
		},
		{
			name:         "empty env yields zero limits",
			env:          map[string]string{},
			want:         Limits{},
			wantWarnings: 0,
		},
		{
			name: "invalid value is ignored and warned",
			env: map[string]string{
				"MEMORY_LIMIT":    "not-a-size",
				"DISK_READ_LIMIT": "100MB",
			},
			want: Limits{
				DiskReadBytes: 100000000,
			},
			wantWarnings: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := &stubLogger{}
			lookup := func(key string) string { return test.env[key] }

			got := LoadLimitsFromEnv(lookup, logger)

			assert.Equal(t, test.want, got)
			assert.Equal(t, test.wantWarnings, logger.warnings)
		})
	}
}

func TestMonitorReportBreaches(t *testing.T) {
	newMonitor := func(collectors ...metricsCollector) *monitor {
		return &monitor{
			resultCh:    make(chan *Metrics, 1),
			breachCh:    make(chan Breach, 1),
			doneChannel: make(chan struct{}),
			logger:      &stubLogger{},
			collectors:  collectors,
		}
	}

	t.Run("forwards the first breach and reports sent", func(t *testing.T) {
		want := Breach{Kind: ResourceMemory, Limit: 100, Usage: 96, Percent: 96}
		m := newMonitor(&fakeCollector{breaches: []Breach{want}})

		assert.True(t, m.reportBreaches())
		select {
		case got := <-m.Breaches():
			assert.Equal(t, want, got)
		default:
			t.Fatal("expected a breach on the channel")
		}
	})

	t.Run("no breach reports not sent and leaves the channel empty", func(t *testing.T) {
		m := newMonitor(&fakeCollector{})

		assert.False(t, m.reportBreaches())
		assert.Len(t, m.Breaches(), 0)
	})
}
