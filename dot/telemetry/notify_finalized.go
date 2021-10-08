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

type NotifyFinalizedTM struct {
	Best *common.Hash `json:"best"`
	// Height is same as block.Header.Number
	Height *big.Int `json:"height"`
	Msg    string   `json:"msg"`
}

func NewNotifyFinalizedTM(best common.Hash, height *big.Int) *NotifyFinalizedTM {
	return &NotifyFinalizedTM{
		Best:   &best,
		Height: height,
		Msg:    "notify.finalized",
	}
}

func (tm *NotifyFinalizedTM) messageType() string {
	return tm.Msg
}
