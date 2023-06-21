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

//go:embed digest.kusama.1482002.zip
var digestKusama1482002Zip []byte

// DigestKusama1482002 returns the bytes of the Kusama digest
// for block 1482002.
func DigestKusama1482002(t *testing.T) (b []byte) {
	bytesReader := bytes.NewReader(digestKusama1482002Zip)

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
