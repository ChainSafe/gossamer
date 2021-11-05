package utils

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	titleRegex = `[A-Za-z]+\([A-Za-z/]+\):.+[A-Za-z]+`
	bodyRegex  = `## Changes.*- .*[A-Za-z0-9].*## Tests.*[A-Za-z].*## Issues.*- .*[A-Za-z0-9].*## Primary Reviewer.*- @.+[A-Za-z0-9].*`
)

var (
	titleRegexp   = regexp.MustCompile(`^[A-Za-z-_]+\([-_/A-Za-z ]+\):.+[A-Za-z]+.+$`)
	commentRegexp = regexp.MustCompile(`<!--(.|\n)*?-->`)
)

// CheckPRDescription verifies the PR title and body match the expected format.
func CheckPRDescription(title, body string) error {
	match, err := regexp.MatchString(titleRegex, title)
	if err != nil || !match {
		return fmt.Errorf("title pattern is not valid: %w match %t", err, match)
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
	bodyData = bodyData + body

	lineSplit := strings.Split(bodyData, "\n")
	joinedLine := strings.Join(lineSplit, "")

	// Regex for body data
	match, err = regexp.MatchString(bodyRegex, joinedLine)
	if err != nil || !match {
		return fmt.Errorf("body pattern is not valid: %w match %t", err, match)
	}
	return nil
}
