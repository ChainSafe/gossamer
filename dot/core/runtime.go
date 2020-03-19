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
	"bytes"
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/lib/babe"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/transaction"

	log "github.com/ChainSafe/log15"
)

// StorageRoot returns the hash of the runtime storage root
func (s *Service) StorageRoot() (common.Hash, error) {
	if s.storageState == nil {
		return common.Hash{}, fmt.Errorf("storage state is nil")
	}
	return s.storageState.StorageRoot()
}

// ValidateTransaction runs the extrinsic through runtime function TaggedTransactionQueue_validate_transaction and returns *Validity
func (s *Service) ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error) {
	ret, err := s.rt.Exec(runtime.TaggedTransactionQueueValidateTransaction, e)
	if err != nil {
		return nil, err
	}

	if ret[0] != 0 {
		return nil, errors.New("could not validate transaction")
	}

	v := transaction.NewValidity(0, [][]byte{{}}, [][]byte{{}}, 0, false)
	_, err = scale.Decode(ret[1:], v)

	return v, err
}

// runs the block through runtime function Core_execute_block
// doesn't return data, but will error if the call isn't successful
func (s *Service) executeBlock(b []byte) error {
	_, err := s.rt.Exec(runtime.CoreExecuteBlock, b)
	if err != nil {
		return err
	}

	return nil
}

// checkForRuntimeChanges checks if changes to the runtime code have occurred; if so, load the new runtime
func (s *Service) checkForRuntimeChanges() error {
	currentCodeHash, err := s.storageState.LoadCodeHash()
	if err != nil {
		return err
	}

	if !bytes.Equal(currentCodeHash[:], s.codeHash[:]) {
		code, err := s.storageState.LoadCode()
		if err != nil {
			return err
		}

		s.rt.Stop()

		s.rt, err = runtime.NewRuntime(code, s.storageState, s.keys)
		if err != nil {
			return err
		}

		// kill babe session, handleBabeSession will reload it with the new runtime
		if s.isAuthority {
			err = s.safeBabeKill()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// TODO: update grandpaAuthorities runtime method, pass latest block number
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
