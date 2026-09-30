package resource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fixture")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestFindCgroupPath(t *testing.T) {
	const content = `12:blkio:/mypath/blkio
11:memory:/mypath/memory
10:cpu,cpuacct:/mypath/cpu
9:pids:/mypath/pids
0::/mypath/unified
`

	tests := []struct {
		name      string
		subsystem string
		want      string
		wantErr   bool
	}{
		{
			name:      "v2 unified hierarchy id",
			subsystem: "0",
			want:      "/mypath/unified",
		},
		{
			name:      "v1 memory subsystem",
			subsystem: "memory",
			want:      "/mypath/memory",
		},
		{
			name:      "subsystem not present",
			subsystem: "devices",
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := findCgroupPath(writeTempFile(t, content), test.subsystem)
			if test.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestFindCgroupPathMissingFile(t *testing.T) {
	_, err := findCgroupPath(filepath.Join(t.TempDir(), "missing"), "memory")
	assert.Error(t, err)
}

func TestReadStatField(t *testing.T) {
	const content = `usage_usec 123456
user_usec 100000
system_usec 23456
`

	tests := []struct {
		name    string
		key     string
		want    uint64
		wantErr bool
	}{
		{
			name: "existing key",
			key:  "usage_usec",
			want: 123456,
		},
		{
			name: "another existing key",
			key:  "system_usec",
			want: 23456,
		},
		{
			name:    "missing key",
			key:     "nonexistent",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := readStatField(writeTempFile(t, content), test.key)
			if test.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestReadStatFieldMissingFile(t *testing.T) {
	_, err := readStatField(filepath.Join(t.TempDir(), "missing"), "usage_usec")
	assert.Error(t, err)
}

func TestReadDiskBytes(t *testing.T) {
	const content = `8:0 rbytes=1024 wbytes=2048 rios=10 wios=20
8:16 rbytes=512 wbytes=256 rios=5 wios=3
`

	read, write, err := readDiskBytes(writeTempFile(t, content))
	require.NoError(t, err)
	assert.Equal(t, uint64(1024+512), read)
	assert.Equal(t, uint64(2048+256), write)

	_, _, err = readDiskBytes(filepath.Join(t.TempDir(), "missing"))
	assert.Error(t, err)
}

func TestSubtractBaseline(t *testing.T) {
	tests := []struct {
		name     string
		current  uint64
		baseline uint64
		want     uint64
	}{
		{
			name:     "normal delta",
			current:  100,
			baseline: 30,
			want:     70,
		},
		{
			name:     "equal values",
			current:  50,
			baseline: 50,
			want:     0,
		},
		{
			name:     "counter reset guarded",
			current:  10,
			baseline: 40,
			want:     0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, subtractBaseline(test.current, test.baseline))
		})
	}
}

func TestResolveCgroupDir(t *testing.T) {
	t.Run("uses joined path when it has the stat file", func(t *testing.T) {
		base := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(base, "sub"), 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(base, "sub", "memory.current"), []byte("1"), 0o600))

		assert.Equal(t, base+"/sub", resolveCgroupDir(base, "/sub", "memory.current"))
	})

	t.Run("falls back to base dir when joined path is missing the stat file", func(t *testing.T) {
		base := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(base, "memory.current"), []byte("1"), 0o600))

		// The /proc/self/cgroup path does not exist under base, so it falls back to the bind-mounted base.
		assert.Equal(t, base, resolveCgroupDir(base, "/docker/deadbeef", "memory.current"))
	})

	t.Run("returns empty when neither location has the stat file", func(t *testing.T) {
		base := t.TempDir()
		assert.Equal(t, "", resolveCgroupDir(base, "/docker/deadbeef", "memory.current"))
	})
}

func TestReadUint64File(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    uint64
		wantErr bool
	}{
		{
			name:    "plain value",
			content: "123456",
			want:    123456,
		},
		{
			name:    "trailing newline trimmed",
			content: "789\n",
			want:    789,
		},
		{
			name:    "non-numeric",
			content: "max",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := readUint64File(writeTempFile(t, test.content))
			if test.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

// writeCgroupFile writes name with content inside dir.
func writeCgroupFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

func TestCgroupCollectorFinalizeAndWrite(t *testing.T) {
	t.Run("reads memory.peak, cpu, and baselined disk", func(t *testing.T) {
		dir := t.TempDir()
		writeCgroupFile(t, dir, cgroupV2MemoryFile, "1000")
		writeCgroupFile(t, dir, cgroupV2MemoryPeakFile, "5000")
		writeCgroupFile(t, dir, cgroupV2IOFile, "8:0 rbytes=1500 wbytes=2500\n")

		c := &cgroupCollector{
			memoryDir:         dir,
			diskDir:           dir,
			diskReadBaseline:  500,
			diskWriteBaseline: 500,
		}

		c.finalizeSample()

		m := &Metrics{}
		c.writeMetrics(m)

		require.NotNil(t, m.PeakMemoryBytes)
		assert.Equal(t, int64(5000), *m.PeakMemoryBytes)

		require.NotNil(t, m.TotalDiskReadBytes)
		assert.Equal(t, int64(1000), *m.TotalDiskReadBytes)
		require.NotNil(t, m.TotalDiskWriteBytes)
		assert.Equal(t, int64(2000), *m.TotalDiskWriteBytes)
	})

	t.Run("working set excludes inactive page cache", func(t *testing.T) {
		dir := t.TempDir()
		writeCgroupFile(t, dir, cgroupV2MemoryFile, "1000")
		writeCgroupFile(t, dir, cgroupV2MemoryStatFile, "anon 300\nfile 700\ninactive_file 600\n")

		c := &cgroupCollector{memoryDir: dir}
		c.finalizeSample()

		assert.Equal(t, uint64(400), c.workingSetBytes)
		assert.Equal(t, uint64(1000), c.peakMemoryBytes)
	})

	t.Run("working set falls back to memory usage when memory.stat is unreadable", func(t *testing.T) {
		dir := t.TempDir()
		writeCgroupFile(t, dir, cgroupV2MemoryFile, "1000")

		c := &cgroupCollector{memoryDir: dir}
		c.finalizeSample()

		assert.Equal(t, uint64(1000), c.workingSetBytes)
	})

	t.Run("falls back to peak of memory.current when memory.peak absent", func(t *testing.T) {
		dir := t.TempDir()
		writeCgroupFile(t, dir, cgroupV2MemoryFile, "3200")

		c := &cgroupCollector{memoryDir: dir}
		c.finalizeSample()

		m := &Metrics{}
		c.writeMetrics(m)

		require.NotNil(t, m.PeakMemoryBytes)
		assert.Equal(t, int64(3200), *m.PeakMemoryBytes)
	})

	t.Run("fallback peak keeps rising across samples when memory.peak absent", func(t *testing.T) {
		dir := t.TempDir()
		writeCgroupFile(t, dir, cgroupV2MemoryFile, "1000")

		c := &cgroupCollector{memoryDir: dir}
		c.finalizeSample() // 1000

		writeCgroupFile(t, dir, cgroupV2MemoryFile, "9000")
		c.finalizeSample() // 9000 must become the new peak, not stay pinned at 1000

		m := &Metrics{}
		c.writeMetrics(m)

		require.NotNil(t, m.PeakMemoryBytes)
		assert.Equal(t, int64(9000), *m.PeakMemoryBytes)
	})

	t.Run("reports nil for unreadable sources", func(t *testing.T) {
		c := &cgroupCollector{memoryDir: t.TempDir(), cpuDir: t.TempDir(), diskDir: t.TempDir()}
		c.finalizeSample()

		m := &Metrics{}
		c.writeMetrics(m)

		assert.Nil(t, m.PeakMemoryBytes)
		assert.Nil(t, m.TotalCPUTimeMS)
		assert.Nil(t, m.TotalDiskReadBytes)
		assert.Nil(t, m.TotalDiskWriteBytes)
	})

	t.Run("cpu delta counts from the primed baseline", func(t *testing.T) {
		dir := t.TempDir()
		writeCgroupFile(t, dir, cgroupV2CPUFile, "usage_usec 13000\n")

		// Emulate newCgroupCollector priming lastCPUMicros so the delta is measured from monitor start.
		c := &cgroupCollector{cpuDir: dir, lastCPUMicros: 10000}
		c.finalizeSample() // 13000 - 10000 = 3000us = 3ms

		m := &Metrics{}
		c.writeMetrics(m)

		require.NotNil(t, m.TotalCPUTimeMS)
		assert.Equal(t, float64(3), *m.TotalCPUTimeMS)
	})

	t.Run("short job records cpu from a zero baseline in a single sample", func(t *testing.T) {
		// A sub-interval job whose primed baseline read 0 must count the full cumulative, not drop it.
		dir := t.TempDir()
		writeCgroupFile(t, dir, cgroupV2CPUFile, "usage_usec 5000\n")

		c := &cgroupCollector{cpuDir: dir, lastCPUMicros: 0}
		c.finalizeSample()

		m := &Metrics{}
		c.writeMetrics(m)

		require.NotNil(t, m.TotalCPUTimeMS)
		assert.Equal(t, float64(5), *m.TotalCPUTimeMS)
	})
}

func TestCgroupCollectorCheckWarnings(t *testing.T) {
	t.Run("disk read warning fires at 80 percent", func(t *testing.T) {
		c := &cgroupCollector{
			limits:            Limits{DiskReadBytes: 100},
			diskReadThreshold: newThresholdTracker(100),
			diskReadBytes:     85, // 85% of limit crosses the 80% level
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 1, logger.warnings)
	})

	t.Run("disk write warning fires from disk write usage", func(t *testing.T) {
		c := &cgroupCollector{
			limits:             Limits{DiskWriteBytes: 100},
			diskWriteThreshold: newThresholdTracker(100),
			diskWriteBytes:     85,
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 1, logger.warnings)
	})

	t.Run("each limit is driven only by its own usage field", func(t *testing.T) {
		// Guards against a threshold being wired to the wrong usage field.
		c := &cgroupCollector{
			limits:             Limits{DiskWriteBytes: 100},
			memThreshold:       newThresholdTracker(0),
			diskReadThreshold:  newThresholdTracker(0),
			diskWriteThreshold: newThresholdTracker(100),
			workingSetBytes:    1 << 30,
			diskReadBytes:      1 << 30,
			diskWriteBytes:     10,
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 0, logger.warnings)
	})

	t.Run("multiple limits crossed at once each warn independently", func(t *testing.T) {
		c := &cgroupCollector{
			limits:             Limits{MemoryBytes: 100, DiskReadBytes: 100, DiskWriteBytes: 100},
			memThreshold:       newThresholdTracker(100),
			diskReadThreshold:  newThresholdTracker(100),
			diskWriteThreshold: newThresholdTracker(100),
			workingSetBytes:    85,
			diskReadBytes:      85,
			diskWriteBytes:     85,
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 3, logger.warnings)
	})

	t.Run("memory warning logs at warning level, not error", func(t *testing.T) {
		c := &cgroupCollector{
			limits:          Limits{MemoryBytes: 100},
			memThreshold:    newThresholdTracker(100),
			workingSetBytes: 85,
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 1, logger.warnings)
		assert.Equal(t, 0, logger.errors)
		assert.Equal(t, 1, logger.flushes)
	})

	t.Run("memory warning ignores a peak inflated by page cache", func(t *testing.T) {
		c := &cgroupCollector{
			limits:          Limits{MemoryBytes: 100},
			memThreshold:    newThresholdTracker(100),
			workingSetBytes: 10,
			peakMemoryBytes: 99,
		}

		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 0, logger.warnings)
	})

	t.Run("memory warning escalates once per level and re-warns past the limit", func(t *testing.T) {
		c := &cgroupCollector{
			limits:       Limits{MemoryBytes: 1000},
			memThreshold: newThresholdTracker(1000),
		}
		logger := &stubLogger{}

		// wantTotal is the cumulative warning count after each per-sample checkWarnings call.
		steps := []struct {
			current   uint64
			wantTotal int
		}{
			{500, 0},  // 50% -- below the first threshold, no warning
			{790, 0},  // 79% -- still below 80%
			{800, 1},  // 80% -- first warning
			{850, 1},  // 85% -- same level, must not re-fire
			{900, 2},  // 90% -- second warning
			{1000, 3}, // 100% -- third warning
			{1100, 3}, // 110% -- below the +25% re-warn step, no new warning
			{1250, 4}, // 125% -- re-warns 25% past the limit
			{1500, 5}, // 150% -- re-warns again
		}
		for _, s := range steps {
			c.workingSetBytes = s.current
			c.checkWarnings(logger)
			assert.Equalf(t, s.wantTotal, logger.warnings, "after current=%d", s.current)
		}
		assert.Equal(t, 0, logger.errors)
	})

	t.Run("no memory warning when limit is unset", func(t *testing.T) {
		c := &cgroupCollector{
			limits:          Limits{MemoryBytes: 0},
			memThreshold:    newThresholdTracker(0),
			workingSetBytes: 1 << 40,
		}
		logger := &stubLogger{}
		c.checkWarnings(logger)

		assert.Equal(t, 0, logger.warnings)
	})
}

func TestCgroupCollectorCheckBreaches(t *testing.T) {
	t.Run("memory breach reports the memory kind from memory usage", func(t *testing.T) {
		c := newCgroupCollector(Limits{MemoryBytes: 100})
		c.workingSetBytes = 96
		breaches := c.checkBreaches()
		require.Len(t, breaches, 1)
		assert.Equal(t, ResourceMemory, breaches[0].Kind)
		assert.Equal(t, uint64(96), breaches[0].Usage)
	})

	t.Run("disk read and write breach independently from their own usage", func(t *testing.T) {
		c := newCgroupCollector(Limits{DiskReadBytes: 100, DiskWriteBytes: 100})
		c.diskReadBytes = 99
		c.diskWriteBytes = 10 // below critical, must not breach
		breaches := c.checkBreaches()
		require.Len(t, breaches, 1)
		assert.Equal(t, ResourceDiskRead, breaches[0].Kind)
	})

	t.Run("no breach below critical", func(t *testing.T) {
		c := newCgroupCollector(Limits{MemoryBytes: 100})
		c.workingSetBytes = 94
		assert.Empty(t, c.checkBreaches())
	})

	t.Run("no breach when limits unset", func(t *testing.T) {
		c := newCgroupCollector(Limits{})
		c.workingSetBytes = 1 << 40
		c.diskReadBytes = 1 << 40
		c.diskWriteBytes = 1 << 40
		assert.Empty(t, c.checkBreaches())
	})
}
