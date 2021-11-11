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

// txpoolImportTM holds `txpool.import` telemetry message, which is supposed to be
// sent when a new transaction gets imported in the transaction pool.
type txpoolImportTM struct {
	Ready  uint `json:"ready"`
	Future uint `json:"future"`
}

// NewTxpoolImportTM creates a new txpoolImportTM struct
func NewTxpoolImportTM(ready, future uint) Message {
	return &txpoolImportTM{
		Ready:  ready,
		Future: future,
	}
}

func (txpoolImportTM) messageType() string {
	return txPoolImportMsg
}
