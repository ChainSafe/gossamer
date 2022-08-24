// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/trie"
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
func NewStateSync(gData *genesis.Data, storageAPI StorageAPI,
	stateVersion trie.Version) (stateSync SyncStateAPI, err error) {
	tmpGen := &genesis.Genesis{
		Genesis: genesis.Fields{
			Raw:     make(map[string]map[string]string),
			Runtime: make(map[string]map[string]interface{}),
		},
	}

	// set genesis fields data
	ent, err := storageAPI.Entries(nil, stateVersion)
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
