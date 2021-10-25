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

package modules

import (
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
)

// GenSyncSpecRequest represents request to get chain specification.
type GenSyncSpecRequest struct {
	Raw bool
}

// SyncStateModule is an RPC module to interact with sync state methods.
type SyncStateModule struct {
	syncStateAPI SyncStateAPI
}

// NewSyncStateModule creates an instance of SyncStateModule given SyncStateAPI.
func NewSyncStateModule(syncStateAPI SyncStateAPI) *SyncStateModule {
	return &SyncStateModule{syncStateAPI: syncStateAPI}
}

// GenSyncSpec returns the JSON serialised chain specification running the node
// (i.e. the current state state), with a sync state.
func (ss *SyncStateModule) GenSyncSpec(_ *http.Request, req *GenSyncSpecRequest, res *genesis.Genesis) error {
	g, err := ss.syncStateAPI.GenSyncSpec(req.Raw)
	if err != nil {
		return err
	}

	*res = *g
	return nil
}

// syncState implements SyncStateAPI.
type syncState struct {
	chainSpecification *genesis.Genesis
}

// NewStateSync creates an instance of SyncStateAPI given a chain specification.
func NewStateSync(gData *genesis.Data, storageAPI StorageAPI) (SyncStateAPI, error) {
	tmpGen := &genesis.Genesis{
		Name:       "",
		ID:         "",
		Bootnodes:  nil,
		ProtocolID: "",
		Genesis: genesis.Fields{
			Runtime: nil,
		},
	}
	tmpGen.Genesis.Raw = make(map[string]map[string]string)
	tmpGen.Genesis.Runtime = make(map[string]map[string]interface{})

	// set genesis fields data
	ent, err := storageAPI.Entries(nil)
	if err != nil {
		return nil, err
	}

	err = genesis.BuildFromMap(ent, tmpGen)
	if err != nil {
		return nil, err
	}

	tmpGen.Name = gData.Name
	tmpGen.ID = gData.ID
	tmpGen.Bootnodes = common.BytesToStringArray(gData.Bootnodes)
	tmpGen.ProtocolID = gData.ProtocolID

	return syncState{chainSpecification: tmpGen}, nil
}

// GenSyncSpec returns the JSON serialised chain specification running the node
// (i.e. the current state), with a sync state.
func (s syncState) GenSyncSpec(raw bool) (*genesis.Genesis, error) {
	if raw {
		err := s.chainSpecification.ToRaw()
		if err != nil {
			return nil, err
		}
	}

	return s.chainSpecification, nil
}
