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
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
)

// blockImportTM struct to hold block import telemetry messages
type blockImportTM struct {
	BestHash *common.Hash `json:"best"`
	Height   *big.Int     `json:"height"`
	Origin   string       `json:"origin"`
}

// NewBlockImportTM function to create new Block Import Telemetry Message
func NewBlockImportTM(bestHash *common.Hash, height *big.Int, origin string) Message {
	return &blockImportTM{
		BestHash: bestHash,
		Height:   height,
		Origin:   origin,
	}
}

func (blockImportTM) messageType() string {
	return blockImportMsg
}
