// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.

// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.

// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package module

import (
	"math/big"

	"github.com/ChainSafe/gossamer/common"
	log "github.com/ChainSafe/log15"
)

type BlocktreeModule struct {
	Blocktree BlocktreeApi
}

// P2pApi is the interface expected to implemented by `p2p` package
type BlocktreeApi interface {
	GetBlockHashOfNode(*big.Int) common.Hash
	LastFinalizedHead() common.Hash
}

func NewBlocktreeModule(blocktreeapi BlocktreeApi) *BlocktreeModule {
	return &BlocktreeModule{blocktreeapi}
}

func (p *BlocktreeModule) GetBlockHashOfNode(num *big.Int) common.Hash {
	log.Debug("[rpc] Executing Chain.getBlockHash", "params", nil)
	return p.Blocktree.GetBlockHashOfNode(num)
}

func (p *BlocktreeModule) LastFinalizedHead() common.Hash {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Blocktree.LastFinalizedHead()
}
