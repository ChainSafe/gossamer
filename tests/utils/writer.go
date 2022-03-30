// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"io"
	"testing"
)

// TestWriter is a writer implementing `io.Writer`
// using the Go test logger `t.Log()`.
type TestWriter struct {
	t *testing.T
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	tw.t.Helper()
	line := string(p)
	tw.t.Log(line)
	return len(p), nil
}

// NewTestWriter creates a new writer which uses
// the Go test logger to write out.
func NewTestWriter(t *testing.T) (writer io.Writer) {
	return &TestWriter{
		t: t,
	}
}
