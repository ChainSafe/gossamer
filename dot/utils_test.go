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

package dot

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// TestNewConfig tests the NewTestConfig method
func TestNewConfig(t *testing.T) {
	cfg := NewTestConfig(t)

	defer utils.RemoveTestDir(t)

	require.NotNil(t, cfg) // TODO: improve dot utils tests
}

// TestNewConfigAndFile tests the NewTestConfigWithFile method
func TestNewConfigAndFile(t *testing.T) {
	testCfg, testCfgFile := NewTestConfigWithFile(t)

	defer utils.RemoveTestDir(t)

	require.NotNil(t, testCfg)     // TODO: improve dot utils tests
	require.NotNil(t, testCfgFile) // TODO: improve dot utils tests
}
