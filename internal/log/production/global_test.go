package production

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GlobalLogger(t *testing.T) {
	t.Parallel()

	buffer := bytes.NewBuffer(nil)

	Patch(SetWriter(buffer))

	Errorf("word %d", 1)

	childLogger := NewFromGlobal(SetLevel(Error))

	childLogger.Error("word 2")

	lines := strings.Split(buffer.String(), "\n")
	buffer.Reset()

	// Check for trailing newline
	require.NotEmpty(t, lines)
	assert.Equal(t, "", lines[len(lines)-1])
	lines = lines[:len(lines)-1]

	expectedRegexes := []string{
		timePrefixRegex + "EROR word 1$",
		timePrefixRegex + "EROR word 2$",
	}

	require.Equal(t, len(expectedRegexes), len(lines))

	for i := range lines {
		regex, err := regexp.Compile(expectedRegexes[i])
		require.NoError(t, err)

		assert.True(t, regex.MatchString(lines[i]),
			"line %q does not match regex %q", lines[i], expectedRegexes[i])
	}
}
