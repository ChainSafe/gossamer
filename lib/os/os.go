// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package os

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EnsureDir ensures the given directory exists, creating it if necessary.
// Errors if the path already exists as a non-directory.
func EnsureDir(dir string, mode os.FileMode) error {
	err := os.MkdirAll(dir, mode)
	if err != nil {
		return fmt.Errorf("could not create directory %q: %w", dir, err)
	}
	return nil
}

// FileExists returns true if the given file exists.
func FileExists(filePath string) bool {
	_, err := os.Stat(filepath.Clean(filePath))
	return !os.IsNotExist(err)
}

// ReadFile reads the given file and returns its contents.
func ReadFile(filePath string) ([]byte, error) {
	return os.ReadFile(filepath.Clean(filePath))
}

// WriteFile writes the given contents to the given file.
func WriteFile(filePath string, contents []byte, mode os.FileMode) error {
	return os.WriteFile(filepath.Clean(filePath), contents, mode)
}

// CopyFile copies a file. It truncates the destination file if it exists.
func CopyFile(src, dst string) error {
	srcfile, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}

	info, err := srcfile.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("cannot read from directories")
	}

	// create new file, truncate if exists and apply same permissions as the original one
	dstfile, err := os.OpenFile(filepath.Clean(dst), os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}

	_, err = io.Copy(dstfile, srcfile)
	if err != nil {
		return fmt.Errorf("could not copy file %q to %q: %w", src, dst, err)
	}

	err = srcfile.Close()
	if err != nil {
		return fmt.Errorf("could not close src file %q: %w", src, err)
	}

	err = dstfile.Close()
	if err != nil {
		return fmt.Errorf("could not close dst file %q: %w", dst, err)
	}

	return nil
}
