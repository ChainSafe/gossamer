// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrTitlePatternNotValid indicates the title does not match the expected pattern.
	ErrTitlePatternNotValid = errors.New("title pattern is not valid")
	// ErrBodySectionNotFound indicates one of the required body section was not found.
	ErrBodySectionNotFound = errors.New("body section not found")
	// ErrBodySectionMisplaced indicates one of the required body section was misplaced in the body.
	ErrBodySectionMisplaced = errors.New("body section misplaced")
)

var (
	titleRegexp   = regexp.MustCompile(`^[A-Za-z-_]+\([-_/A-Za-z ]+\):.+[A-Za-z]+.+$`)
	commentRegexp = regexp.MustCompile(`<!--(.|\n)*?-->`)
)

// CheckPRDescription verifies the PR title and body match the expected format.
func CheckPRDescription(title, body string) error {
	if !titleRegexp.MatchString(title) {
		return fmt.Errorf("%w: for regular expression %s: '%s'",
			ErrTitlePatternNotValid, titleRegexp.String(), title)
	}

	body = commentRegexp.ReplaceAllString(body, "")
	body = strings.ReplaceAll(body, "\r", "")

	// Required subheading sections in order
	requiredSections := []string{"Changes", "Tests", "Issues", "Primary Reviewer"}

	previousIndex := -1
	previousSection := ""
	for i, requiredSection := range requiredSections {
		textToFind := "## " + requiredSection
		if i > 0 {
			// no new line required before the first section
			textToFind = "\n" + textToFind
		}
		if i < len(requiredSections)-1 {
			// no new line required for last section
			textToFind += "\n"
		}

		index := strings.Index(body, textToFind)
		if index == -1 {
			body = strings.ReplaceAll(body, "\n", "\\n") // for error logs in one line
			return fmt.Errorf("%w: %q in body: %s", ErrBodySectionNotFound, textToFind, body)
		} else if i > 0 && index < previousIndex {
			return fmt.Errorf("%w: section %q cannot be before section %q",
				ErrBodySectionMisplaced, requiredSection, previousSection)
		}
		previousIndex = index
		previousSection = requiredSection
	}

	return nil
}
