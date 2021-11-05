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
