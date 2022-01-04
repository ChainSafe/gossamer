// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package testdata

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed data.3b1b0.zip
var data3b1b0Zip []byte

// Data3b1b0 returns the bytes of the network data starting with
// 0x3b1b0.
func Data3b1b0(t *testing.T) (b []byte) {
	bytesReader := bytes.NewReader(data3b1b0Zip)

	zipReader, err := zip.NewReader(bytesReader, int64(bytesReader.Len()))
	require.NoError(t, err)

	require.Len(t, zipReader.File, 1)
	zipFile := zipReader.File[0]

	file, err := zipFile.Open()
	require.NoError(t, err)

	defer func() {
		err := file.Close()
		require.NoError(t, err)
	}()

	b, err = io.ReadAll(file)
	require.NoError(t, err)

	return b
}
