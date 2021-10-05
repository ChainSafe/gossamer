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
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/ChainSafe/gossamer/lib/genesis"
)

type SyncStateModule struct {
	SyncStateAPI SyncStateAPI
}

type BoolRequest struct {
	Raw bool
}

func NewSyncStateModule(s SyncStateAPI) *SyncStateModule {
	return &SyncStateModule{SyncStateAPI: s}
}

func (ss *SyncStateModule) GenSyncSpec(_ *http.Request, req *BoolRequest, res *genesis.Genesis) error {
	genesis, err := ss.SyncStateAPI.GenSyncSpec(req.Raw)
	if err != nil {
		return err
	}

	*res = *genesis
	return nil
}

type SyncState struct {
	GenesisFilePath string
}

// GenSyncSpec returns the JSON serialized chain specification running the node
// (i.e. the current state), with a sync state.
func (s SyncState) GenSyncSpec(raw bool) (*genesis.Genesis, error) {
	fp, err := filepath.Abs(s.GenesisFilePath)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	g := new(genesis.Genesis)
	err = json.Unmarshal(data, g)
	if err != nil {
		return nil, err
	}

	if raw {
		err = g.ToRaw()
		if err != nil {
			return nil, err
		}
	}

	return g, nil
}
