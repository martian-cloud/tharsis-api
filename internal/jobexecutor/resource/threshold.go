package resource

// warningPercents are the limit-usage percentages at which a warning fires, in ascending order.
var warningPercents = []int{80, 90, 100}

const (
	// rewarnStepPercent re-warns every additional 25% past 100%, since sustained overrun is the actionable signal.
	rewarnStepPercent = 25

	// criticalPercent sits below 100% to leave headroom before the kernel enforces the limit itself.
	criticalPercent = 95
)

// thresholdTracker fires as usage crosses 80/90/100% of a limit, then re-warns every rewarnStepPercent beyond 100%.
type thresholdTracker struct {
	limit       uint64
	nextPercent int
}

func newThresholdTracker(limit uint64) thresholdTracker {
	tw := thresholdTracker{limit: limit}
	if limit > 0 {
		tw.nextPercent = warningPercents[0]
	}

	return tw
}

// check returns true and the crossed percentage the first time usage reaches the next warning level.
func (tw *thresholdTracker) check(current uint64) (bool, int) {
	if tw.limit == 0 || tw.nextPercent == 0 {
		return false, 0
	}

	percent := int(current * 100 / tw.limit)
	if percent < tw.nextPercent {
		return false, 0
	}

	crossed := tw.nextPercent
	tw.nextPercent = nextWarningPercent(crossed)

	return true, crossed
}

// breach reports a Breach at criticalPercent of a non-zero limit; unlike check, it is stateless and safe to call every tick.
func (tw *thresholdTracker) breach(kind Kind, usage uint64) (Breach, bool) {
	if tw.limit == 0 {
		return Breach{}, false
	}

	percent := int(usage * 100 / tw.limit)
	if percent < criticalPercent {
		return Breach{}, false
	}

	return Breach{Kind: kind, Limit: tw.limit, Usage: usage, Percent: percent}, true
}

// nextWarningPercent returns the next level after crossing the given one, stepping by rewarnStepPercent once past 100%.
func nextWarningPercent(crossed int) int {
	for _, p := range warningPercents {
		if p > crossed {
			return p
		}
	}

	return crossed + rewarnStepPercent
}
