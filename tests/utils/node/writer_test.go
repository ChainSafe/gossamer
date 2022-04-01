package node

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_prefixedWriter(t *testing.T) {
	t.Parallel()

	writer := bytes.NewBuffer(nil)
	prefixWriter := &prefixedWriter{
		prefix: []byte("prefix: "),
		writer: writer,
	}

	message := []byte("message\n")
	n, err := prefixWriter.Write(message)
	require.NoError(t, err)
	expectedBytesWrittenCount := 16
	assert.Equal(t, expectedBytesWrittenCount, n)
	expectedWritten := "prefix: message\n"
	assert.Equal(t, expectedWritten, writer.String())

	message = []byte("message two\n")
	n, err = prefixWriter.Write(message)
	require.NoError(t, err)
	expectedBytesWrittenCount = 20
	assert.Equal(t, expectedBytesWrittenCount, n)
	expectedWritten = "prefix: message\nprefix: message two\n"
	assert.Equal(t, expectedWritten, writer.String())
}
