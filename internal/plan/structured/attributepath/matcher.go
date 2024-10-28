// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

// Package attributepath provides a way to match paths for replace and sensitive attributes
package attributepath

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Matcher provides an interface for stepping through changes following an
// attribute path.
//
// GetChildWithKey and GetChildWithIndex will check if any of the internal paths
// match the provided key or index, and return a new Matcher that will match
// that children or potentially it's children.
//
// The caller of the above functions is required to know whether the next value
// in the path is a list type or an object type and call the relevant function,
// otherwise these functions will crash/panic.
//
// The Matches function returns true if the paths you have traversed until now
// ends.
type Matcher interface {
	// Matches returns true if we have reached the end of a path and found an
	// exact match.
	Matches() bool

	// MatchesPartial returns true if the current attribute is part of a path
	// but not necessarily at the end of the path.
	MatchesPartial() bool

	GetChildWithKey(key string) Matcher
	GetChildWithIndex(index int) (Matcher, error)
}

// Parse accepts a json.RawMessage and outputs a formatted Matcher object.
//
// Parse expects the message to be a JSON array of JSON arrays containing
// strings and floats. This function happily accepts a null input representing
// none of the changes in this resource are causing a replacement. The propagate
// argument tells the matcher to propagate any matches to the matched attributes
// children.
//
// In general, this function is designed to accept messages that have been
// produced by the lossy cty.Paths conversion functions within the tjson
// package. There is nothing particularly special about that conversion process
// though, it just produces the nested JSON arrays described above.
func Parse(paths []interface{}, propagate bool) (Matcher, error) {
	matcher := &PathMatcher{
		Propagate: propagate,
	}

	for _, path := range paths {
		nestedPaths, ok := path.([]interface{})
		if !ok {
			return nil, errors.New("wrong replace path type")
		}
		matcher.Paths = append(matcher.Paths, nestedPaths)
	}

	return matcher, nil
}

// Empty returns an empty PathMatcher that will by default match nothing.
//
// We give direct access to the PathMatcher struct so a matcher can be built
// in parts with the Append and AppendSingle functions.
func Empty(propagate bool) *PathMatcher {
	return &PathMatcher{
		Propagate: propagate,
	}
}

// Append accepts an existing PathMatcher and returns a new one that attaches
// all the paths from message with the existing paths.
//
// The new PathMatcher is created fresh, and the existing one is unchanged.
func Append(matcher *PathMatcher, message json.RawMessage) (*PathMatcher, error) {
	var values [][]interface{}
	if err := json.Unmarshal(message, &values); err != nil {
		return nil, errors.New("failed to unmarshal attribute paths: " + err.Error())
	}

	return &PathMatcher{
		Propagate: matcher.Propagate,
		Paths:     append(matcher.Paths, values...),
	}, nil
}

// AppendSingle accepts an existing PathMatcher and returns a new one that
// attaches the single path from message with the existing paths.
//
// The new PathMatcher is created fresh, and the existing one is unchanged.
func AppendSingle(matcher *PathMatcher, messages []json.RawMessage) (*PathMatcher, error) {
	values := []interface{}{}

	for _, message := range messages {
		var value []interface{}
		if err := json.Unmarshal(message, &value); err != nil {
			return nil, errors.New("failed to unmarshal attribute paths: " + err.Error())
		}
		values = append(values, value)
	}

	return &PathMatcher{
		Propagate: matcher.Propagate,
		Paths:     append(matcher.Paths, values),
	}, nil
}

// PathMatcher contains a slice of paths that represent paths through the values
// to relevant/tracked attributes.
type PathMatcher struct {
	// We represent our internal paths as a [][]interface{} as the cty.Paths
	// conversion process is lossy. Since the type information is lost there
	// is no (easy) way to reproduce the original cty.Paths object. Instead,
	// we simply rely on the external callers to know the type information and
	// call the correct GetChild function.
	Paths [][]interface{}

	// Propagate tells the matcher that it should propagate any matches it finds
	// onto the children of that match.
	Propagate bool
}

// Matches returns true if there is a match
func (p *PathMatcher) Matches() bool {
	for _, path := range p.Paths {
		if len(path) == 0 {
			return true
		}
	}
	return false
}

// MatchesPartial returns true if there is a partial match
func (p *PathMatcher) MatchesPartial() bool {
	return len(p.Paths) > 0
}

// GetChildWithKey returns a new matcher with the child key if it's found
func (p *PathMatcher) GetChildWithKey(key string) Matcher {
	child := &PathMatcher{
		Propagate: p.Propagate,
	}
	for _, path := range p.Paths {
		if len(path) == 0 {
			// This means that the current value matched, but not necessarily
			// it's child.

			if p.Propagate {
				// If propagate is true, then our child match our matches
				child.Paths = append(child.Paths, path)
			}

			// If not we would simply drop this path from our set of paths but
			// either way we just continue.
			continue
		}

		if path[0].(string) == key {
			child.Paths = append(child.Paths, path[1:])
		}
	}
	return child
}

// GetChildWithIndex returns a new matcher with the child index if it's found
func (p *PathMatcher) GetChildWithIndex(index int) (Matcher, error) {
	child := &PathMatcher{
		Propagate: p.Propagate,
	}
	for _, path := range p.Paths {
		if len(path) == 0 {
			// This means that the current value matched, but not necessarily
			// it's child.

			if p.Propagate {
				// If propagate is true, then our child match our matches
				child.Paths = append(child.Paths, path)
			}

			// If not we would simply drop this path from our set of paths but
			// either way we just continue.
			continue
		}

		// Terraform actually allows user to provide strings into indexes as
		// long as the string can be interpreted into a number. For example, the
		// following are equivalent and we need to support them.
		//    - test_resource.resource.list[0].attribute
		//    - test_resource.resource.list["0"].attribute
		//
		// Note, that Terraform will raise a validation error if the string
		// can't be coerced into a number, so we will panic here if anything
		// goes wrong safe in the knowledge the validation should stop this from
		// happening.

		switch val := path[0].(type) {
		case float64:
			if int(path[0].(float64)) == index {
				child.Paths = append(child.Paths, path[1:])
			}
		case string:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, fmt.Errorf("found invalid type within path (%v:%T), the validation shouldn't have allowed this to happen; this is a bug in Terraform, please report it", val, val)
			}
			if int(f) == index {
				child.Paths = append(child.Paths, path[1:])
			}
		default:
			return nil, fmt.Errorf("found invalid type within path (%v:%T), the validation shouldn't have allowed this to happen; this is a bug in Terraform, please report it", val, val)
		}
	}
	return child, nil
}

// AlwaysMatcher returns a matcher that will always match all paths.
func AlwaysMatcher() Matcher {
	return &alwaysMatcher{}
}

type alwaysMatcher struct{}

func (a *alwaysMatcher) Matches() bool {
	return true
}

func (a *alwaysMatcher) MatchesPartial() bool {
	return true
}

func (a *alwaysMatcher) GetChildWithKey(_ string) Matcher {
	return a
}

func (a *alwaysMatcher) GetChildWithIndex(_ int) (Matcher, error) {
	return a, nil
}
