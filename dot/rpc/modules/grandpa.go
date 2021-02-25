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
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
)

// GrandpaModule init parameters
type GrandpaModule struct {
	blockAPI BlockAPI
}

// NewGrandpaModule creates a new Grandpa rpc module.
func NewGrandpaModule(api BlockAPI) *GrandpaModule {
	return &GrandpaModule{
		blockAPI: api,
	}
}

// ProveFinalityRequest request struct
type ProveFinalityRequest struct {
	blockHashStart common.Hash
	blockHashEnd   common.Hash
	authorityID    uint64
}

// ProveFinalityResponse is an optional SCALE encoded proof array
type ProveFinalityResponse [][]byte

// ProveFinality for the provided block range. Returns NULL if there are no known finalized blocks in the range. If no authorities set is provided, the current one will be attempted.
func (gm *GrandpaModule) ProveFinality(r *http.Request, req *ProveFinalityRequest, res *ProveFinalityResponse) error {
	blocksToCheck, err := gm.blockAPI.SubChain(req.blockHashStart, req.blockHashEnd)
	if err != nil {
		return err
	}

	// Leaving check in for linter
	if req.authorityID != uint64(0) {
		// TODO: #1404 Check if functionality relevant
	}

	for _, block := range blocksToCheck {
		hasJustification, _ := gm.blockAPI.HasJustification(block)
		if !hasJustification {
			continue
		}

		justification, err := gm.blockAPI.GetJustification(block)
		if err != nil {
			continue
		}
		*res = append(*res, justification)
	}

	return nil
}
