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
var testMetaData []byte

func init() {
	bytesReader := bytes.NewReader(testMetaData)
	zipReader, err := zip.NewReader(bytesReader, int64(bytesReader.Len()))
	if err != nil {
		panic(err)
	}

	zipFile := zipReader.File[0]

	file, err := zipFile.Open()
	if err != nil {
		panic(err)
	}

	defer func() {
		err := file.Close()
		if err != nil {
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

// GetTestData returns the testData string.
func GetTestData() string {
	return testData
}
