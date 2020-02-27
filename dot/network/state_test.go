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
	"math/big"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// MockBlockState ...
type MockBlockState struct {
	number *big.Int
}

func newMockBlockState(number *big.Int) *MockBlockState {
	return &MockBlockState{number: number}
}

// LatestHeader for MockBlockState
func (mbs *MockBlockState) LatestHeader() *types.Header {
	parentHash, err := common.HexToHash("0x4545454545454545454545454545454545454545454545454545454545454545")
	if err != nil {
		return nil
	}
	stateRoot, err := common.HexToHash("0xb3266de137d20a5d0ff3a6401eb57127525fd9b2693701f0bf5a8a853fa3ebe0")
	if err != nil {
		return nil
	}
	extrinsicsRoot, err := common.HexToHash("0x03170a2e7597b7b7e3d84c05391d139a62b157e78786d8c082f29dcf4c111314")
	if err != nil {
		return nil
	}

	return &types.Header{
		ParentHash:     parentHash,
		Number:         mbs.number,
		StateRoot:      stateRoot,
		ExtrinsicsRoot: extrinsicsRoot,
		Digest:         [][]byte{{}},
	}
}

// MockNetworkState ...
type MockNetworkState struct{}
