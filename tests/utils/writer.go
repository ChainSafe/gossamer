package utils

import (
	"io"
	"testing"
)

type TestWriter struct {
	t *testing.T
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	tw.t.Helper()
	line := string(p)
	tw.t.Log(line)
	return len(p), nil
}

func NewTestWriter(t *testing.T) (writer io.Writer) {
	return &TestWriter{
		t: t,
	}
}
