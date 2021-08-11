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

package digest

import (
	"math/big"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/grandpa"
)

// BlockState interface for block state methods
type BlockState interface {
	BestBlockHeader() (*types.Header, error)
	RegisterImportedChannel(ch chan<- *types.Block) (byte, error)
	UnregisterImportedChannel(id byte)
	RegisterFinalizedChannel(ch chan<- *types.FinalisationInfo) (byte, error)
	UnregisterFinalisedChannel(id byte)
}

// EpochState is the interface for state.EpochState
type EpochState interface {
	GetEpochForBlock(header *types.Header) (uint64, error)
	SetEpochData(epoch uint64, info *types.EpochData) error
	SetConfigData(epoch uint64, info *types.ConfigData) error
}

// GrandpaState is the interface for the state.GrandpaState
type GrandpaState interface {
	SetNextChange(authorities []*grandpa.Voter, number *big.Int) error
	IncrementSetID() error
	SetNextPause(number *big.Int) error
	SetNextResume(number *big.Int) error
	GetCurrentSetID() (uint64, error)
}
