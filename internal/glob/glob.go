// Package glob matches values against go-glob patterns and checks whether one pattern's matches
// fully contain another's.
package glob

import (
	"strings"

	goglob "github.com/ryanuber/go-glob"
)

// Shadows reports whether every value that b matches also matches a, conservatively so it can
// miss an unusual but genuinely contained pattern but never claims containment that isn't guaranteed.
func Shadows(a, b string) bool {
	if a == "*" {
		return true
	}

	if b == "*" {
		return false
	}

	if !strings.Contains(a, "*") {
		return a == b
	}

	if !strings.Contains(b, "*") {
		return Glob(a, b)
	}

	leadA, trailA, segsA := splitSegments(a)
	leadB, trailB, segsB := splitSegments(b)

	if len(segsA) == 0 {
		return true
	}

	if len(segsB) == 0 {
		return false
	}

	return segmentsContained(leadA, trailA, segsA, leadB, trailB, segsB)
}

// Glob reports whether value matches pattern under go-glob's own syntax.
func Glob(pattern, value string) bool {
	return goglob.Glob(pattern, value)
}

// blockPosition names where in b's segments a match for one of a's segments was found.
type blockPosition struct {
	block  int
	offset int
}

// segmentsContained reports whether every string shaped like b also matches a's own shape.
func segmentsContained(leadA, trailA bool, segsA []string, leadB, trailB bool, segsB []string) bool {
	pos := blockPosition{}

	for i, seg := range segsA {
		if i == 0 && !leadA {
			if leadB || !strings.HasPrefix(segsB[0], seg) {
				return false
			}

			pos = blockPosition{block: 0, offset: len(seg)}

			continue
		}

		if i == len(segsA)-1 && !trailA {
			if trailB {
				return false
			}

			lastBlock := segsB[len(segsB)-1]
			minOffset := 0
			if pos.block == len(segsB)-1 {
				minOffset = pos.offset
			}

			if len(lastBlock)-len(seg) < minOffset || !strings.HasSuffix(lastBlock, seg) {
				return false
			}

			continue
		}

		found, ok := locateSegment(segsB, pos, seg)
		if !ok {
			return false
		}

		pos = found
	}

	return true
}

// locateSegment finds the earliest occurrence of seg in segsB at or after from.
func locateSegment(segsB []string, from blockPosition, seg string) (blockPosition, bool) {
	for block := from.block; block < len(segsB); block++ {
		start := 0
		if block == from.block {
			start = from.offset
		}

		if start > len(segsB[block]) {
			continue
		}

		if idx := strings.Index(segsB[block][start:], seg); idx >= 0 {
			return blockPosition{block: block, offset: start + idx + len(seg)}, true
		}
	}

	return blockPosition{}, false
}

// splitSegments breaks pattern into the literal segments go-glob matches in order.
func splitSegments(pattern string) (leading, trailing bool, segments []string) {
	leading = strings.HasPrefix(pattern, "*")
	trailing = strings.HasSuffix(pattern, "*")

	for _, part := range strings.Split(pattern, "*") {
		if part != "" {
			segments = append(segments, part)
		}
	}

	return leading, trailing, segments
}
