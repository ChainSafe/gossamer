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

package blocktree

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/core"
)

var zeroHash, _ = common.HexToHash("0x00")

func createGenesisBlock() core.Block {
	return core.Block{
		SlotNumber:   nil,
		PreviousHash: zeroHash,
		//VrfOutput:    nil,
		//Transactions: nil,
		//Signature:    nil,
		BlockNumber: nil,
		Hash:        common.Hash{0x00},
	}
}

func TestBlockTree_AddBlock_GetBlock(t *testing.T) {
	bt := NewBlockTreeFromGenesis(createGenesisBlock())

	oneHash, _ := common.HexToHash("0x01")
	twoHash, _ := common.HexToHash("0x02")

	block1 := core.Block{
		PreviousHash: zeroHash,
		Hash:         oneHash,
		BlockNumber:  big.NewInt(1),
	}

	bt.AddBlock(block1)

	block2 := core.Block{
		PreviousHash: oneHash,
		Hash:         twoHash,
		BlockNumber:  big.NewInt(2),
	}

	bt.AddBlock(block2)

	blk := bt.GetNode(twoHash)

	if blk.number.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("got: %s expected: %s", blk.number, big.NewInt(2))
	}

	fmt.Println(bt.String())
}

