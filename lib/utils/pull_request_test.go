// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CheckPRDescription(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		title string
		body  string
		err   error
	}{
		"all empty": {
			err: errors.New(`title pattern is not valid: for regular expression ^[A-Za-z-_]+\([-_/A-Za-z ]+\):.+[A-Za-z]+.+$: ''`),
		},
		"invalid title": {
			title: "category: something",
			err:   errors.New(`title pattern is not valid: for regular expression ^[A-Za-z-_]+\([-_/A-Za-z ]+\):.+[A-Za-z]+.+$: 'category: something'`),
		},
		"empty body only": {
			title: "category(subcategory): something",
			err:   errors.New("body section not found: \"## Changes\\n\" in body: "),
		},
		"invalid body": {
			title: "category(subcategory): something",
			body:  "##Change\n## Tests ## Issues ## Primary Reviewer",
			err:   errors.New("body section not found: \"## Changes\\n\" in body: ##Change\\n## Tests ## Issues ## Primary Reviewer"),
		},
		"misplaced section": {
			title: "category(subcategory): something",
			body:  "## Changes\n## Tests\n## Primary Reviewer\n## Issues\n",
			err:   errors.New("body section misplaced: section \"Primary Reviewer\" cannot be before section \"Issues\""),
		},
		"minimal valid": {
			title: "category(subcategory): something",
			body:  "## Changes\n## Tests\n## Issues\n## Primary Reviewer",
		},
		"valid example": {
			title: `feat(dot/rpc): implement chain_subscribeAllHeads RPC`,
			body: `## Changes

<!--
Please provide a brief but specific list of changes made, describe the changes
in functionality rather than the changes in code.
-->

- changes for demo :123

## Tests

<!--
Details on how to run tests relevant to the changes within this pull request.
-->

- tests for demo:123{}

## Issues

<!--
Please link any issues that this pull request is related to and use the GitHub
supported format for automatically closing issues (ie, closes #123, fixes #123)
-->

- issues for demo:43434

## Primary Reviewer

<!--Please indicate one of the code owners that are required to review prior to merging changes (e.g. @noot)
-->

- @noot for demo:12
`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := CheckPRDescription(testCase.title, testCase.body)
			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
