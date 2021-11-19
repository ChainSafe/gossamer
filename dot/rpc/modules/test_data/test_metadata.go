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
		return
	}

	zipFile := zipReader.File[0]

	file, err := zipFile.Open()
	if err != nil {
		return
	}

	defer func() {
		err := file.Close()
		if err != nil {
			return
		}
	}()

	b, err := io.ReadAll(file)
	if err != nil {
		return
	}

	testData = string(b)
}

var testData string

// GetTestData returns the testData string.
func GetTestData() string {
	return testData
}
