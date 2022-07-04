// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"context"
	"fmt"
	"os/exec"

	libutils "github.com/ChainSafe/gossamer/lib/utils"
)

// BuildGossamer finds the project root path and builds the Gossamer
// binary to ./bin/gossamer at the project root path.
func BuildGossamer() (err error) {
	rootPath, err := libutils.GetProjectRootPath()
	if err != nil {
		return fmt.Errorf("get project root path: %w", err)
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "build",
		"-trimpath", "-o", "./bin/gossamer", "./cmd/gossamer")
	cmd.Dir = rootPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("building Gossamer: %w\n%s", err, output)
	}

	return nil
}
