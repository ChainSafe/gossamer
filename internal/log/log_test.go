package log

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Logger_Debug(t *testing.T) {
	t.Parallel()

	buffer := bytes.NewBuffer(nil)

	logger := New(SetLevel(LevelDebug), SetWriter(buffer), SetCaller(CallerShort))

	logger.Debug("isn't this \"function\"...")
	logger.Debug("...fun?")

	result := buffer.String()
	buffer.Reset()

	result = strings.TrimSuffix(result, "\n")

	lines := strings.Split(result, "\n")
	require.Len(t, lines, 2)

	expectedVariablePrefix := regexp.MustCompile(`2[0-9]{3}/[0-1][0-9]/[0-3][0-9] [0-2][0-9]:[0-5][0-9]:[0-5][0-9] `)

	expectedLinesWithoutPrefix := []string{
		`log_test.go:20: DBUG isn't this "function"...`,
		`log_test.go:21: DBUG ...fun?`,
	}

	for i, line := range lines {
		prefix := expectedVariablePrefix.FindString(line)
		assert.NotEmpty(t, prefix)
		line = line[len(prefix):]
		assert.Equal(t, expectedLinesWithoutPrefix[i], line)
	}
}
