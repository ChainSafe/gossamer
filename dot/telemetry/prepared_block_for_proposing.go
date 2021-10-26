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

// preparedBlockForProposingTM holds a 'prepared_block_for_proposing' telemetry
// message, which is supposed to be sent when a new block is built.
type preparedBlockForProposingTM struct {
	Hash common.Hash `json:"hash"`
	// Height of the chain, Block.Header.Number
	Number string `json:"number"`
	Msg    string `json:"msg"`
}

// NewPreparedBlockForProposingTM gets a new PreparedBlockForProposingTM struct.
func NewPreparedBlockForProposingTM(hash common.Hash, number string) Message {
	return &preparedBlockForProposingTM{
		Hash:   hash,
		Number: number,
		Msg:    "prepared_block_for_proposing",
	}
}

func (tm *preparedBlockForProposingTM) messageType() string {
	return tm.Msg
}
