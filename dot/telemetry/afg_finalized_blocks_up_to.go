// Copyright 2021 ChainSafe Systems (ON) Corp.
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

package telemetry

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// afgFinalizedBlocksUpToTM holds telemetry message of type `afg.finalized_blocks_up_to`,
// which is supposed to be send GRANDPA client finalises new blocks.
type afgFinalizedBlocksUpToTM struct {
	Hash   common.Hash `json:"hash"`
	Number string      `json:"number"`
}

// NewAfgFinalizedBlocksUpToTM creates a new afgFinalizedBlocksUpToTM struct.
func NewAfgFinalizedBlocksUpToTM(hash common.Hash, number string) Message {
	return &afgFinalizedBlocksUpToTM{
		Hash:   hash,
		Number: number,
	}
}

func (afgFinalizedBlocksUpToTM) messageType() string {
	return afgFinalizedBlocksUpToMsg
}
