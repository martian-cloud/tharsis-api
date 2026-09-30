package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThresholdTrackerCheck(t *testing.T) {
	tests := []struct {
		name        string
		limit       uint64
		readings    []uint64
		wantWarned  []bool
		wantCrossed []int
	}{
		{
			name:        "no limit never warns",
			limit:       0,
			readings:    []uint64{50, 100, 200},
			wantWarned:  []bool{false, false, false},
			wantCrossed: []int{0, 0, 0},
		},
		{
			name:        "below first level",
			limit:       100,
			readings:    []uint64{79},
			wantWarned:  []bool{false},
			wantCrossed: []int{0},
		},
		{
			name:        "crosses 80 then 90 then 100",
			limit:       100,
			readings:    []uint64{80, 90, 100},
			wantWarned:  []bool{true, true, true},
			wantCrossed: []int{80, 90, 100},
		},
		{
			name:        "each level fires once",
			limit:       100,
			readings:    []uint64{80, 85, 90},
			wantWarned:  []bool{true, false, true},
			wantCrossed: []int{80, 0, 90},
		},
		{
			name:        "jump straight past 80 to 90 reports 80 first",
			limit:       100,
			readings:    []uint64{95},
			wantWarned:  []bool{true},
			wantCrossed: []int{80},
		},
		{
			name:        "same reading advances one level per call",
			limit:       100,
			readings:    []uint64{100, 100, 100},
			wantWarned:  []bool{true, true, true},
			wantCrossed: []int{80, 90, 100},
		},
		{
			name:        "re-warns beyond 100 every 25 percent",
			limit:       100,
			readings:    []uint64{80, 90, 100, 125, 150},
			wantWarned:  []bool{true, true, true, true, true},
			wantCrossed: []int{80, 90, 100, 125, 150},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracker := newThresholdTracker(test.limit)
			for i, reading := range test.readings {
				warned, crossed := tracker.check(reading)
				assert.Equal(t, test.wantWarned[i], warned, "reading %d (%d)", i, reading)
				assert.Equal(t, test.wantCrossed[i], crossed, "reading %d (%d)", i, reading)
			}
		})
	}
}

func TestThresholdTrackerBreach(t *testing.T) {
	tests := []struct {
		name        string
		limit       uint64
		usage       uint64
		wantOK      bool
		wantPercent int
	}{
		{name: "zero limit never breaches", limit: 0, usage: 1 << 40, wantOK: false},
		{name: "below critical does not breach", limit: 100, usage: 94, wantOK: false},
		{name: "exactly critical breaches", limit: 100, usage: 95, wantOK: true, wantPercent: 95},
		{name: "over critical breaches with actual percent", limit: 100, usage: 130, wantOK: true, wantPercent: 130},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tracker := newThresholdTracker(test.limit)
			b, ok := tracker.breach(ResourceMemory, test.usage)
			assert.Equal(t, test.wantOK, ok)
			if ok {
				assert.Equal(t, ResourceMemory, b.Kind)
				assert.Equal(t, test.limit, b.Limit)
				assert.Equal(t, test.usage, b.Usage)
				assert.Equal(t, test.wantPercent, b.Percent)
			}
		})
	}
}

func TestNextWarningPercent(t *testing.T) {
	tests := []struct {
		name    string
		crossed int
		want    int
	}{
		{
			name:    "80 to 90",
			crossed: 80,
			want:    90,
		},
		{
			name:    "90 to 100",
			crossed: 90,
			want:    100,
		},
		{
			name:    "100 steps by 25",
			crossed: 100,
			want:    125,
		},
		{
			name:    "125 steps by 25",
			crossed: 125,
			want:    150,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, nextWarningPercent(test.crossed))
		})
	}
}
