package utils

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	// ErrTitlePatternNotValid indicates the title does not match the expected pattern.
	ErrTitlePatternNotValid = errors.New("title pattern is not valid")
	// ErrBodyPatternNotValid indicates the body does not match the expected pattern.
	ErrBodyPatternNotValid = errors.New("body pattern is not valid")
)

var (
	titleRegexp   = regexp.MustCompile(`^[A-Za-z]+\([A-Za-z/]+\):.+[A-Za-z]+$`)
	bodyRegexp    = regexp.MustCompile(`^(.|\n)*## Changes\n+(.|\n)+\n+## Tests\n+(.|\n)+\n+## Issues\n+(.|\n)+\n+## Primary Reviewer\n+(.|\n)+$`)
	commentRegexp = regexp.MustCompile(`<!--(.|\n)*?-->`)
)

// CheckPRDescription verifies the PR title and body match the expected format.
func CheckPRDescription(title, body string) error {
	if !titleRegexp.MatchString(title) {
		return fmt.Errorf("%w: for regular expression %s: '%s'",
			ErrTitlePatternNotValid, titleRegexp.String(), title)
	}

	body = commentRegexp.ReplaceAllString(body, "")

	if !bodyRegexp.MatchString(body) {
		return fmt.Errorf("%w: for regular expression %s: '%s'",
			ErrBodyPatternNotValid, bodyRegexp.String(), body)
	}

	return nil
}
