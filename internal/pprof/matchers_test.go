// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import (
	"regexp"

	"go.uber.org/mock/gomock"
)

var _ gomock.Matcher = (*regexMatcher)(nil)

type regexMatcher struct {
	regexp *regexp.Regexp
}

func (r *regexMatcher) Matches(x interface{}) bool {
	s, ok := x.(string)
	if !ok {
		return false
	}
	return r.regexp.MatchString(s)
}

func (r *regexMatcher) String() string {
	return "regular expression " + r.regexp.String()
}

func newRegexMatcher(regex string) *regexMatcher {
	return &regexMatcher{
		regexp: regexp.MustCompile(regex),
	}
}
