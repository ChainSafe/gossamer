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
		"valid example 2": {
			title: `fix(dot/state): fix deadlock, fixes bootstrap syncing`,
			body: `## Changes

<!--

Please provide a brief but specific list of changes made, describe the changes
in functionality rather than the changes in code.

-->

- deadlock was caused as ` + "`" + `GetBlockByHash()` + "`" + ` calls ` + "`" + `RLock()` + "`" + `, ` + "`" + `GetBlockByHash()` + "`" + ` calls ` + "`" + `GetBlockBody()` + "`" + ` which previously also called ` + "`" + `RLock()` + "`" + `. this was ok when there were no calls to the write lock ` + "`" + `Lock()` + "`" + ` simultaneously. however the deadlock scenario occurred when one goroutine was calling  ` + "`" + `GetBlockBody()` + "`" + ` (with 1 read-lock obtained) and another called ` + "`" + `SetFinalisedHash()` + "`" + ` (thus obtaining the write-lock). as a read-lock can no longer be obtained if 1. write-lock was obtained (even if it was freed) and 2. another read-lock was already obtained, this caused a deadlock
- I ended up just removing the lock in ` + "`" + `GetBlockBody()` + "`" + ` to prevent multiple read-locks from being obtained. if anyone has a better method let me know :D

## Tests

<!--

Details on how to run tests relevant to the changes within this pull request.

-->

` + "```" + `
make gossamer
./bin/gossamer --chain polkadot
` + "```" + `

node can sync (up until it hits max memory :p) no more stalling/deadlocks

## Issues

<!--

Please link any issues that this pull request is related to and use the GitHub
supported format for automatically closing issues (ie, closes #123, fixes #123)

-->

- related to #1479 

## Primary Reviewer

<!--
Please indicate one of the code owners that are required to review prior to merging changes (e.g. @noot)
-->

- @EclesioMeloJunior 
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
