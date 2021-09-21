package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_PR_Checks(t *testing.T) {
	tests := []struct {
		title string
		body  string
		valid bool
	}{
		{
			title: "",
			body:  "",
			valid: false,
		},
		{
			title: "abc(abc): abc",
			body:  "",
			valid: false,
		},
		{
			title: `feat(dot/rpc): implement chain_subscribeAllHeads RPC`,
			body:  `## Changes\n\n<!--\nPlease provide a brief but specific list of changes made, describe the changes\nin functionality rather than the changes in code.\n-->\n\n- changes for demo :123\n\n## Tests\n\n<!--\nDetails on how to run tests relevant to the changes within this pull request.\n-->\n\n- tests for demo:123{}\n\n## Issues\n\n<!--\nPlease link any issues that this pull request is related to and use the GitHub\nsupported format for automatically closing issues (ie, closes #123, fixes #123)\n-->\n\n- issues for demo:43434\n\n## Primary Reviewer\n\n<!--\nPlease indicate one of the code owners that are required to review prior to merging changes (e.g. @noot)\n-->\n\n- @noot for demo:12`,
			valid: true,
		},
		{
			title: "abc(): abc",
			body:  "",
			valid: false,
		},
		{
			title: "(abc): abc",
			body:  "",
			valid: false,
		},
		{
			title: "abc(abc):",
			body:  "",
			valid: false,
		},
	}

	for _, test := range tests {
		err := CheckPRDescription(test.title, test.body)
		if test.valid {
			require.NoError(t, err, "title", test.title, "body", test.body)
		} else {
			require.Error(t, err, "title", test.title, "body", test.body)
		}
	}
}
