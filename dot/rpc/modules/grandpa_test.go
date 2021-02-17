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

package modules

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
)

func TestGrandpaProveFinality(t *testing.T) {
	state := newTestStateService(t)
	gmSvc := NewGrandpaModule(nil, state.Block)

	blockHash1, _ := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	blockHash2, _ := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	state.Block.SetJustification(blockHash1, make([]byte, 10))
	state.Block.SetJustification(blockHash2, make([]byte, 11))

	res := new(ProveFinalityResponse)
	gmSvc.ProveFinality(nil, &ProveFinalityRequest{
		blockHashStart: blockHash1,
		blockHashEnd:   blockHash2,
	}, res)

	println(res)

	// expected := &ProveFinalityResponse{
	// 	ParentHash:     header.ParentHash.String(),
	// 	Number:         common.BytesToHex(header.Number.Bytes()),
	// 	StateRoot:      header.StateRoot.String(),
	// 	ExtrinsicsRoot: header.ExtrinsicsRoot.String(),
	// 	Digest:         ChainBlockHeaderDigest{},
	// }

}
