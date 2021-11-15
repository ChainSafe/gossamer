// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/urfave/cli"
)

func beginProfile(ctx *cli.Context) (func(), error) {
	cpuStopFunc, err := cpuProfile(ctx)
	if err != nil {
		return nil, err
	}

	memStopFunc, err := memProfile(ctx)
	if err != nil {
		return nil, err
	}

	return func() {
		if cpuStopFunc != nil {
			cpuStopFunc()
		}

		if memStopFunc != nil {
			memStopFunc()
		}
	}, nil
}

func cpuProfile(ctx *cli.Context) (func(), error) {
	cpuProfFile := ctx.GlobalString(CPUProfFlag.Name)
	if cpuProfFile == "" {
		return nil, nil
	}

	cpuFile, err := os.Create(cpuProfFile)
	if err != nil {
		return nil, err
	}

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		return nil, err
	}

	return func() {
		pprof.StopCPUProfile()
		if err := cpuFile.Close(); err != nil {
			logger.Error("failed to close file " + cpuFile.Name())
		}
	}, nil
}

func memProfile(ctx *cli.Context) (func(), error) {
	memProfFile := ctx.GlobalString(MemProfFlag.Name)
	if memProfFile == "" {
		return nil, nil
	}

	memFile, err := os.Create(memProfFile)
	if err != nil {
		return nil, err
	}

	return func() {
		runtime.GC()
		if err := pprof.WriteHeapProfile(memFile); err != nil {
			logger.Errorf("could not write memory profile: %s", err)
		}

		if err := memFile.Close(); err != nil {
			logger.Error("failed to close file " + memFile.Name())
		}
	}, nil
}
