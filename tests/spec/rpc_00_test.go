// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package spec

import (
	"fmt"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/tests/rpc"
)

const (
	currentPort = "8540"
)

func TestMain(m *testing.M) {
	if rpc.GOSSAMER_INTEGRATION_TEST_MODE != "rpc_spec" {
		_, _ = fmt.Fprintln(os.Stdout, "Going to skip RPC spec tests")
		return
	}

	_, _ = fmt.Fprintln(os.Stdout, "Going to start RPC spec test")

	// Start all tests
	code := m.Run()
	os.Exit(code)
}
