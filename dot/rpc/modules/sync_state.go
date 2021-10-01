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
	"io/ioutil"
	"path/filepath"
)

type SyncState struct {
	GenesisFilePath string
}

// GenSyncSpec returns the JSON serialized chain specification running the node
// (i.e. the current state), with a sync state.
func (s SyncState) GenSyncSpec(raw bool) ([]byte, error) {
	fp, err := filepath.Abs(s.GenesisFilePath)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	return data, nil
}
