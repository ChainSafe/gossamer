// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package testdata

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"io"
)

//go:embed test_metadata.zip
var testRuntimeMetaData []byte

func init() {
	bytesReader := bytes.NewReader(testRuntimeMetaData)
	zipReader, err := zip.NewReader(bytesReader, int64(bytesReader.Len()))
	if err != nil {
		panic(err)
	} else if len(zipReader.File) == 0 {
		panic("no file in test_metadata.zip")
	}

	zipFile := zipReader.File[0]

	file, err := zipFile.Open()
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			panic(err)
		}
	}()

	b, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	testData = string(b)
}

var testData string

// NewTestMetadata returns the testData string.
func NewTestMetadata() string {
	return testData
}
