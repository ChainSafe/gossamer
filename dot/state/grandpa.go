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

package state

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
)

var (
	genesisSetID      = uint64(0)
	grandpaPrefix     = "grandpa"
	authoritiesPrefix = []byte("auth")
	setIDChangePrefix = []byte("change")
	pauseKey          = []byte("pause")
	resumeKey         = []byte("resume")
	currentSetIDKey   = []byte("setID")
)

// GrandpaState tracks information related to grandpa
type GrandpaState struct {
	db chaindb.Database
}

// NewGrandpaStateFromGenesis returns a new GrandpaState given the grandpa genesis authorities
func NewGrandpaStateFromGenesis(db chaindb.Database, genesisAuthorities []types.GrandpaVoter) (*GrandpaState, error) {
	grandpaDB := chaindb.NewTable(db, grandpaPrefix)
	s := &GrandpaState{
		db: grandpaDB,
	}

	if err := s.setCurrentSetID(genesisSetID); err != nil {
		return nil, err
	}

	if err := s.SetLatestRound(0); err != nil {
		return nil, err
	}

	if err := s.setAuthorities(genesisSetID, genesisAuthorities); err != nil {
		return nil, err
	}

	if err := s.setSetIDChangeAtBlock(genesisSetID, big.NewInt(0)); err != nil {
		return nil, err
	}

	return s, nil
}

// NewGrandpaState returns a new GrandpaState
func NewGrandpaState(db chaindb.Database) (*GrandpaState, error) {
	return &GrandpaState{
		db: chaindb.NewTable(db, grandpaPrefix),
	}, nil
}

func authoritiesKey(setID uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, setID)
	return append(authoritiesPrefix, buf...)
}

func setIDChangeKey(setID uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, setID)
	return append(setIDChangePrefix, buf...)
}

// setAuthorities sets the authorities for a given setID
func (s *GrandpaState) setAuthorities(setID uint64, authorities []types.GrandpaVoter) error {
	enc, err := scale.Encode(authorities)
	if err != nil {
		return err
	}

	return s.db.Put(authoritiesKey(setID), enc)
}

// GetAuthorities returns the authorities for the given setID
func (s *GrandpaState) GetAuthorities(setID uint64) ([]types.GrandpaVoter, error) {
	enc, err := s.db.Get(authoritiesKey(setID))
	if err != nil {
		return nil, err
	}

	r := &bytes.Buffer{}
	_, err = r.Write(enc)
	if err != nil {
		return nil, err
	}

	v, err := types.DecodeGrandpaVoters(r)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// setCurrentSetID sets the current set ID
func (s *GrandpaState) setCurrentSetID(setID uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, setID)
	return s.db.Put(currentSetIDKey, buf)
}

// GetCurrentSetID retrieves the current set ID
func (s *GrandpaState) GetCurrentSetID() (uint64, error) {
	id, err := s.db.Get(currentSetIDKey)
	if err != nil {
		return 0, err
	}

	if len(id) < 8 {
		return 0, errors.New("invalid setID")
	}

	return binary.LittleEndian.Uint64(id), nil
}

// SetLatestRound sets the latest finalised GRANDPA round in the db
func (s *GrandpaState) SetLatestRound(round uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, round)
	return s.db.Put(common.LatestFinalizedRoundKey, buf)
}

// GetLatestRound gets the latest finalised GRANDPA roundfrom the db
func (s *GrandpaState) GetLatestRound() (uint64, error) {
	r, err := s.db.Get(common.LatestFinalizedRoundKey)
	if err != nil {
		return 0, err
	}

	round := binary.LittleEndian.Uint64(r[:8])
	return round, nil
}

// SetNextChange sets the next authority change
func (s *GrandpaState) SetNextChange(authorities []types.GrandpaVoter, number *big.Int) error {
	currSetID, err := s.GetCurrentSetID()
	if err != nil {
		return err
	}

	nextSetID := currSetID + 1
	err = s.setAuthorities(nextSetID, authorities)
	if err != nil {
		return err
	}

	err = s.setSetIDChangeAtBlock(nextSetID, number)
	if err != nil {
		return err
	}

	return nil
}

// IncrementSetID increments the set ID
func (s *GrandpaState) IncrementSetID() error {
	currSetID, err := s.GetCurrentSetID()
	if err != nil {
		return err
	}

	nextSetID := currSetID + 1
	return s.setCurrentSetID(nextSetID)
}

// setSetIDChangeAtBlock sets a set ID change at a certain block
func (s *GrandpaState) setSetIDChangeAtBlock(setID uint64, number *big.Int) error {
	return s.db.Put(setIDChangeKey(setID), number.Bytes())
}

// GetSetIDChange returs the block number where the set ID was updated
func (s *GrandpaState) GetSetIDChange(setID uint64) (*big.Int, error) {
	num, err := s.db.Get(setIDChangeKey(setID))
	if err != nil {
		return nil, err
	}

	return big.NewInt(0).SetBytes(num), nil
}

// GetSetIDByBlockNumber returns the set ID for a given block number
func (s *GrandpaState) GetSetIDByBlockNumber(num *big.Int) (uint64, error) {
	curr, err := s.GetCurrentSetID()
	if err != nil {
		return 0, err
	}

	for {
		changeUpper, err := s.GetSetIDChange(curr + 1)
		if err == chaindb.ErrKeyNotFound {
			if curr == 0 {
				return 0, nil
			}
			curr = curr - 1
			continue
		}
		if err != nil {
			return 0, err
		}

		changeLower, err := s.GetSetIDChange(curr)
		if err != nil {
			return 0, err
		}

		// if the given block number is greater or equal to the block number of the set ID change,
		// return the current set ID
		if num.Cmp(changeUpper) < 1 && num.Cmp(changeLower) == 1 {
			return curr, nil
		}

		if num.Cmp(changeUpper) == 1 {
			return curr + 1, nil
		}

		curr = curr - 1

		if int(curr) < 0 {
			return 0, nil
		}
	}
}

// SetNextPause sets the next grandpa pause at the given block number
func (s *GrandpaState) SetNextPause(number *big.Int) error {
	return s.db.Put(pauseKey, number.Bytes())
}

// GetNextPause returns the block number of the next grandpa pause, nil if there is no upcoming pause
func (s *GrandpaState) GetNextPause() (*big.Int, error) {
	num, err := s.db.Get(pauseKey)
	if err == chaindb.ErrKeyNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return big.NewInt(0).SetBytes(num), nil
}

// SetNextResume sets the next grandpa resume at the given block number
func (s *GrandpaState) SetNextResume(number *big.Int) error {
	return s.db.Put(resumeKey, number.Bytes())
}

// GetNextResume returns the block number of the next grandpa resume, nil if there is no upcoming resume
func (s *GrandpaState) GetNextResume() (*big.Int, error) {
	num, err := s.db.Get(resumeKey)
	if err == chaindb.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return big.NewInt(0).SetBytes(num), nil
}

func prevotesKey(round, setID uint64) []byte {
	prevotesPrefix := []byte("pv")
	k := roundAndSetIDToBytes(round, setID)
	return append(prevotesPrefix, k...)
}

func precommitsKey(round, setID uint64) []byte {
	precommitsPrefix := []byte("pc")
	k := roundAndSetIDToBytes(round, setID)
	return append(precommitsPrefix, k...)
}

func roundAndSetIDToBytes(round, setID uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, round)
	buf2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf2, setID)
	return append(buf, buf2...)
}

// SetPrevotes sets the prevotes for a specific round and set ID in the database
func (s *GrandpaState) SetPrevotes(round, setID uint64, pvs []*types.GrandpaSignedVote) error {
	data, err := scale.Encode(pvs)
	if err != nil {
		return err
	}

	return s.db.Put(prevotesKey(round, setID), data)
}

// GetPrevotes retrieves the prevotes for a specific round and set ID from the database
func (s *GrandpaState) GetPrevotes(round, setID uint64) ([]*types.GrandpaSignedVote, error) {
	data, err := s.db.Get(prevotesKey(round, setID))
	if err != nil {
		return nil, err
	}

	pvs, err := scale.Decode(data, []*types.GrandpaSignedVote{})
	if err != nil {
		return nil, err
	}

	return pvs.([]*types.GrandpaSignedVote), nil
}

// SetPrecommits sets the precommits for a specific round and set ID in the database
func (s *GrandpaState) SetPrecommits(round, setID uint64, pcs []*types.GrandpaSignedVote) error {
	data, err := scale.Encode(pcs)
	if err != nil {
		return err
	}

	return s.db.Put(precommitsKey(round, setID), data)
}

// GetPrecommits retrieves the precommits for a specific round and set ID from the database
func (s *GrandpaState) GetPrecommits(round, setID uint64) ([]*types.GrandpaSignedVote, error) {
	data, err := s.db.Get(precommitsKey(round, setID))
	if err != nil {
		return nil, err
	}

	pcs, err := scale.Decode(data, []*types.GrandpaSignedVote{})
	if err != nil {
		return nil, err
	}

	return pcs.([]*types.GrandpaSignedVote), nil
}
