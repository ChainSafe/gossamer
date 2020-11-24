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

package network

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

// TestRequestedBlockIDs tests adding and removing block ids from requestedBlockIDs
func TestRequestedBlockIDs(t *testing.T) {
	basePath := utils.NewTestBasePath(t, "node")

	// removes all data directories created within test directory
	defer utils.RemoveTestDir(t)

	config := &Config{
		BasePath:    basePath,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMDNS:      true,
	}

	node := createTestService(t, config)

	hasRequestedBlockID := node.requestTracker.hasRequestedBlockID(1)
	require.Equal(t, false, hasRequestedBlockID)

	node.requestTracker.addRequestedBlockID(1)

	hasRequestedBlockID = node.requestTracker.hasRequestedBlockID(1)
	require.Equal(t, true, hasRequestedBlockID)

	node.requestTracker.removeRequestedBlockID(1)

	hasRequestedBlockID = node.requestTracker.hasRequestedBlockID(1)
	require.Equal(t, false, hasRequestedBlockID)
}
