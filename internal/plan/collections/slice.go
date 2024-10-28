// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package collections

import (
	"errors"
	"reflect"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/computed"
)

// TransformIndices is a callback function that transforms a pair of indices
type TransformIndices func(before, after int) (computed.Diff, error)

// ProcessIndices is a callback function that processes a pair of indices
type ProcessIndices func(before, after int) error

// IsObjType is a callback function that returns true if the input is an object type
type IsObjType[Input any] func(input Input) (bool, error)

// TransformSlice transforms a pair of slices into a computed.Diff and a action.Action
func TransformSlice[Input any](before, after []Input, process TransformIndices, isObjType IsObjType[Input]) ([]computed.Diff, action.Action, error) {
	current := action.NoOp
	if before != nil && after == nil {
		current = action.Delete
	}
	if before == nil && after != nil {
		current = action.Create
	}

	var elements []computed.Diff
	if err := ProcessSlice(before, after, func(before, after int) error {
		element, err := process(before, after)
		if err != nil {
			return err
		}
		elements = append(elements, element)
		current = CompareActions(current, element.Action)
		return nil
	}, isObjType); err != nil {
		return nil, action.NoOp, err
	}
	return elements, current, nil
}

// ProcessSlice will process each element in the before and after slices
func ProcessSlice[Input any](before, after []Input, process ProcessIndices, isObjType IsObjType[Input]) error {
	lcs, err := LongestCommonSubsequence(before, after, func(before, after Input) bool {
		return reflect.DeepEqual(before, after)
	})
	if err != nil {
		return err
	}

	var beforeIx, afterIx, lcsIx int
	for beforeIx < len(before) || afterIx < len(after) || lcsIx < len(lcs) {
		// Step through all the before values until we hit the next item in the
		// longest common subsequence. We are going to just say that all of
		// these have been deleted.
		for beforeIx < len(before) && (lcsIx >= len(lcs) || !reflect.DeepEqual(before[beforeIx], lcs[lcsIx])) {
			isBeforeObjType, err := isObjType(before[beforeIx])
			if err != nil {
				return err
			}

			isAfterObjType := false
			if afterIx < len(after) {
				typ, err := isObjType(after[afterIx])
				if err != nil {
					return err
				}
				isAfterObjType = typ
			}

			isObjectDiff := isBeforeObjType && isAfterObjType && (lcsIx >= len(lcs) || !reflect.DeepEqual(after[afterIx], lcs[lcsIx]))
			if isObjectDiff {
				if err := process(beforeIx, afterIx); err != nil {
					return err
				}
				beforeIx++
				afterIx++
				continue
			}

			if err := process(beforeIx, len(after)); err != nil {
				return err
			}
			beforeIx++
		}

		// Now, step through all the after values until hit the next item in the
		// LCS. We are going to say that all of these have been created.
		for afterIx < len(after) && (lcsIx >= len(lcs) || !reflect.DeepEqual(after[afterIx], lcs[lcsIx])) {
			if err := process(len(before), afterIx); err != nil {
				return err
			}
			afterIx++
		}

		// Finally, add the item in common as unchanged.
		if lcsIx < len(lcs) {
			if err := process(beforeIx, afterIx); err != nil {
				return err
			}
			beforeIx++
			afterIx++
			lcsIx++
		}
	}

	return nil
}

// LongestCommonSubsequence finds a sequence of values that are common to both
// x and y, with the same relative ordering as in both collections. This result
// is useful as a first step towards computing a diff showing added/removed
// elements in a sequence.
//
// The approached used here is a "naive" one, assuming that both xs and ys will
// generally be small in most reasonable Terraform configurations. For larger
// lists the time/space usage may be sub-optimal.
//
// A pair of lists may have multiple longest common subsequences. In that
// case, the one selected by this function is undefined.
func LongestCommonSubsequence[V any](xs, ys []V, equals func(x, y V) bool) ([]V, error) {
	if len(xs) == 0 || len(ys) == 0 {
		return make([]V, 0), nil
	}

	c := make([]int, len(xs)*len(ys))
	eqs := make([]bool, len(xs)*len(ys))
	w := len(xs)

	for y := 0; y < len(ys); y++ {
		for x := 0; x < len(xs); x++ {
			eq := false
			if equals(xs[x], ys[y]) {
				eq = true
				eqs[(w*y)+x] = true // equality tests can be expensive, so cache it
			}
			if eq {
				// Sequence gets one longer than for the cell at top left,
				// since we'd append a new item to the sequence here.
				if x == 0 || y == 0 {
					c[(w*y)+x] = 1
				} else {
					c[(w*y)+x] = c[(w*(y-1))+(x-1)] + 1
				}
			} else {
				// We follow the longest of the sequence above and the sequence
				// to the left of us in the matrix.
				l := 0
				u := 0
				if x > 0 {
					l = c[(w*y)+(x-1)]
				}
				if y > 0 {
					u = c[(w*(y-1))+x]
				}
				if l > u {
					c[(w*y)+x] = l
				} else {
					c[(w*y)+x] = u
				}
			}
		}
	}

	// The bottom right cell tells us how long our longest sequence will be
	seq := make([]V, c[len(c)-1])

	// Now we will walk back from the bottom right cell, finding again all
	// of the equal pairs to construct our sequence.
	x := len(xs) - 1
	y := len(ys) - 1
	i := len(seq) - 1

	for x > -1 && y > -1 {
		if eqs[(w*y)+x] {
			// Add the value to our result list and then walk diagonally
			// up and to the left.
			seq[i] = xs[x]
			x--
			y--
			i--
		} else {
			// Take the path with the greatest sequence length in the matrix.
			l := 0
			u := 0
			if x > 0 {
				l = c[(w*y)+(x-1)]
			}
			if y > 0 {
				u = c[(w*(y-1))+x]
			}
			if l > u {
				x--
			} else {
				y--
			}
		}
	}

	if i > -1 {
		// should never happen if the matrix was constructed properly
		return nil, errors.New("not enough elements in sequence")
	}

	return seq, nil
}
