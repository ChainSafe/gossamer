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

// CheckPRDescription matches the PR title and body according to the PR template.
func CheckPRDescription(title, body string) error {
	match, err := regexp.MatchString(titleRegex, title)
	if err != nil || !match {
		return fmt.Errorf("title pattern is not valid: %w match %t", err, match)
	}

	var bodyData string
	// Remove comment from PR body.
	for {
		start := strings.Index(body, "<!--")
		end := strings.Index(body, "-->")
		if start < 0 || end < 0 {
			break
		}

		bodyData = bodyData + body[:start]
		body = body[end+4:]
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
