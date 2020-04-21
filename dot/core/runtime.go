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

package core

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"

	log "github.com/ChainSafe/log15"
)

// TODO: move this to runtime package, requires separate babetypes package for AuthorityData
// TODO: this seems to be out-of-date, the call is now named Grandpa_authorities and takes a block number.
func (s *Service) grandpaAuthorities() ([]*babe.AuthorityData, error) {
	ret, err := s.rt.Exec(runtime.AuraAPIAuthorities, []byte{})
	if err != nil {
		return nil, err
	}

	decodedKeys, err := scale.Decode(ret, [][32]byte{})
	if err != nil {
		return nil, err
	}

	keys := decodedKeys.([][32]byte)
	authsRaw := make([]*babe.AuthorityDataRaw, len(keys))

	for i, key := range keys {
		log.Debug("[core] babe authority", "key", fmt.Sprintf("0x%x", key))
		authsRaw[i] = &babe.AuthorityDataRaw{
			ID:     key,
			Weight: 1,
		}
	}

	auths := make([]*babe.AuthorityData, len(keys))
	for i, auth := range authsRaw {
		auths[i] = new(babe.AuthorityData)
		err = auths[i].FromRaw(auth)
		if err != nil {
			return nil, err
		}
	}

	return auths, err
}
