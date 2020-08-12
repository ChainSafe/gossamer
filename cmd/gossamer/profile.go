// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"os"
	"runtime/pprof"

	"github.com/urfave/cli"
)

func cpuProfile(ctx *cli.Context) (func(), error) {
	proffile := ctx.GlobalString(CPUProfFlag.Name)
	if proffile == "" {
		return nil, nil
	}

	f, err := os.Create(proffile)
	if err != nil {
		return nil, err
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		return nil, err
	}

	return func() {
		f.Close()
		pprof.StopCPUProfile()
	}, nil
}
