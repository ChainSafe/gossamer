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

package grandpa

import (
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// BlockState is the interface required by GRANDPA into the block state
type BlockState interface {
	HasHeader(hash common.Hash) (bool, error)
	GetHeader(hash common.Hash) (*types.Header, error)
	IsDescendantOf(parent, child common.Hash) (bool, error)
	HighestCommonAncestor(a, b common.Hash) (common.Hash, error)
	GetFinalizedHeader() (*types.Header, error)
	SetFinalizedHash(hash common.Hash) error
	BestBlockHeader() (*types.Header, error)
	Leaves() []common.Hash
}
