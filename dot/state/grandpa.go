// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/database"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/cockroachdb/pebble"
)

var (
	errPendingScheduledChanges = errors.New("pending scheduled changes needs to be applied")
	errDuplicateHashes         = errors.New("duplicated hashes")
	errAlreadyHasForcedChange  = errors.New("already has a forced change")
	errUnfinalizedAncestor     = errors.New("unfinalized ancestor")

	ErrNoNextAuthorityChange = errors.New("no next authority change")
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
	db         GetPutDeleter
	blockState *BlockState

	forcedChanges        *orderedPendingChanges
	scheduledChangeRoots *changeTree
	telemetry            Telemetry
}

// NewGrandpaStateFromGenesis returns a new GrandpaState given the grandpa genesis authorities
func NewGrandpaStateFromGenesis(db database.Database, bs *BlockState,
	genesisAuthorities []types.GrandpaVoter, telemetry Telemetry) (*GrandpaState, error) {
	grandpaDB := database.NewTable(db, grandpaPrefix)
	s := &GrandpaState{
		db:                   grandpaDB,
		blockState:           bs,
		scheduledChangeRoots: new(changeTree),
		forcedChanges:        new(orderedPendingChanges),
		telemetry:            telemetry,
	}

	if err := s.setCurrentSetID(genesisSetID); err != nil {
		return nil, fmt.Errorf("cannot set current set id: %w", err)
	}

	if err := s.SetLatestRound(0); err != nil {
		return nil, fmt.Errorf("cannot set latest round: %w", err)
	}

	if err := s.setAuthorities(genesisSetID, genesisAuthorities); err != nil {
		return nil, fmt.Errorf("cannot set authorities: %w", err)
	}

	if err := s.setChangeSetIDAtBlock(genesisSetID, 0); err != nil {
		return nil, fmt.Errorf("cannot set change set id at block 0: %w", err)
	}

	return s, nil
}

// NewGrandpaState returns a new GrandpaState
func NewGrandpaState(db database.Database, bs *BlockState, telemetry Telemetry) *GrandpaState {
	return &GrandpaState{
		db:                   database.NewTable(db, grandpaPrefix),
		blockState:           bs,
		scheduledChangeRoots: new(changeTree),
		forcedChanges:        new(orderedPendingChanges),
		telemetry:            telemetry,
	}
}

// HandleGRANDPADigest receives a decoded GRANDPA digest and calls the right function to handles the digest
func (s *GrandpaState) HandleGRANDPADigest(header *types.Header, digest scale.VaryingDataType) error {
	digestValue, err := digest.Value()
	if err != nil {
		return fmt.Errorf("getting digest value: %w", err)
	}
	switch val := digestValue.(type) {
	case types.GrandpaScheduledChange:
		return s.addScheduledChange(header, val)
	case types.GrandpaForcedChange:
		return s.addForcedChange(header, val)
	case types.GrandpaOnDisabled:
		return nil
	case types.GrandpaPause:
		logger.Warn("GRANDPA Pause consensus message not implemented yet")
		return nil
	case types.GrandpaResume:
		logger.Warn("GRANDPA Resume consensus message not implemented yet")
		return nil
	default:
		return fmt.Errorf("not supported digest")
	}
}

func (s *GrandpaState) addForcedChange(header *types.Header, fc types.GrandpaForcedChange) error {
	auths, err := types.GrandpaAuthoritiesRawToAuthorities(fc.Auths)
	if err != nil {
		return fmt.Errorf("cannot parse GRANDPA authorities to raw authorities: %w", err)
	}

	pendingChange := pendingChange{
		bestFinalizedNumber: fc.BestFinalizedBlock,
		nextAuthorities:     auths,
		announcingHeader:    header,
		delay:               fc.Delay,
	}

	err = s.forcedChanges.importChange(pendingChange, s.blockState.IsDescendantOf)
	if err != nil {
		return fmt.Errorf("cannot import forced change: %w", err)
	}

	logger.Debugf("there are now %d possible forced changes", s.forcedChanges.Len())
	return nil
}

func (s *GrandpaState) addScheduledChange(header *types.Header, sc types.GrandpaScheduledChange) error {
	auths, err := types.GrandpaAuthoritiesRawToAuthorities(sc.Auths)
	if err != nil {
		return fmt.Errorf("cannot parse GRANPDA authorities to raw authorities: %w", err)
	}

	pendingChange := &pendingChange{
		nextAuthorities:  auths,
		announcingHeader: header,
		delay:            sc.Delay,
	}

	err = s.scheduledChangeRoots.importChange(pendingChange, s.blockState.IsDescendantOf)
	if err != nil {
		return fmt.Errorf("cannot import scheduled change: %w", err)
	}

	logger.Debugf("there are now %d possible scheduled change roots", s.scheduledChangeRoots.Len())
	return nil
}

// ApplyScheduledChanges will check the schedules changes in order to find a root
// equal or behind the finalized number and will apply its authority set changes
func (s *GrandpaState) ApplyScheduledChanges(finalizedHeader *types.Header) error {
	finalizedHash := finalizedHeader.Hash()

	err := s.forcedChanges.pruneChanges(finalizedHash, s.blockState.IsDescendantOf)
	if err != nil {
		return fmt.Errorf("cannot prune non-descendant forced changes: %w", err)
	}

	if s.scheduledChangeRoots.Len() == 0 {
		return nil
	}

	changeToApply, err := s.scheduledChangeRoots.findApplicable(finalizedHash,
		finalizedHeader.Number, s.blockState.IsDescendantOf)
	if err != nil {
		return fmt.Errorf("cannot get applicable scheduled change: %w", err)
	}

	if changeToApply == nil {
		return nil
	}

	logger.Debugf("applying scheduled change: %s", changeToApply.change)

	newSetID, err := s.IncrementSetID()
	if err != nil {
		return fmt.Errorf("cannot increment set id: %w", err)
	}

	grandpaVotersAuthorities := types.NewGrandpaVotersFromAuthorities(changeToApply.change.nextAuthorities)
	err = s.setAuthorities(newSetID, grandpaVotersAuthorities)
	if err != nil {
		return fmt.Errorf("cannot set authorities: %w", err)
	}

	err = s.setChangeSetIDAtBlock(newSetID, changeToApply.change.effectiveNumber())
	if err != nil {
		return fmt.Errorf("cannot set the change set id at block: %w", err)
	}

	logger.Debugf("Applying authority set change scheduled at block #%d",
		changeToApply.change.announcingHeader.Number)

	canonHeightString := strconv.FormatUint(uint64(changeToApply.change.announcingHeader.Number), 10)
	s.telemetry.SendMessage(telemetry.NewAfgApplyingScheduledAuthoritySetChange(
		canonHeightString,
	))

	return nil
}

// ApplyForcedChanges will check for if there is a scheduled forced change relative to the
// imported block and then apply it otherwise nothing happens
func (s *GrandpaState) ApplyForcedChanges(importedBlockHeader *types.Header) error {
	forcedChange, err := s.forcedChanges.findApplicable(importedBlockHeader.Hash(),
		importedBlockHeader.Number, s.blockState.IsDescendantOf)
	if err != nil {
		return fmt.Errorf("cannot find applicable forced change: %w", err)
	} else if forcedChange == nil {
		return nil
	}

	forcedChangeHash := forcedChange.announcingHeader.Hash()
	bestFinalizedNumber := forcedChange.bestFinalizedNumber

	dependant, err := s.scheduledChangeRoots.lookupChangeWhere(func(pcn *pendingChangeNode) (bool, error) {
		if pcn.change.effectiveNumber() > uint(bestFinalizedNumber) {
			return false, nil
		}

		scheduledBlockHash := pcn.change.announcingHeader.Hash()
		return s.blockState.IsDescendantOf(scheduledBlockHash, forcedChangeHash)
	})
	if err != nil {
		return fmt.Errorf("cannot check pending changes while applying forced change: %w", err)
	} else if dependant != nil {
		return fmt.Errorf("%w: %s", errPendingScheduledChanges, dependant.change)
	}

	logger.Debugf("Applying authority set forced change: %s", forcedChange)

	canonHeightString := strconv.FormatUint(uint64(forcedChange.announcingHeader.Number), 10)
	s.telemetry.SendMessage(telemetry.NewAfgApplyingForcedAuthoritySetChange(
		canonHeightString,
	))

	currentSetID, err := s.GetCurrentSetID()
	if err != nil {
		return fmt.Errorf("cannot get current set id: %w", err)
	}

	err = s.setChangeSetIDAtBlock(currentSetID, uint(forcedChange.bestFinalizedNumber))
	if err != nil {
		return fmt.Errorf("cannot set change set id at block: %w", err)
	}

	newSetID, err := s.IncrementSetID()
	if err != nil {
		return fmt.Errorf("cannot increment set id: %w", err)
	}

	grandpaVotersAuthorities := types.NewGrandpaVotersFromAuthorities(forcedChange.nextAuthorities)
	err = s.setAuthorities(newSetID, grandpaVotersAuthorities)
	if err != nil {
		return fmt.Errorf("cannot set authorities: %w", err)
	}

	err = s.setChangeSetIDAtBlock(newSetID, forcedChange.effectiveNumber())
	if err != nil {
		return fmt.Errorf("cannot set change set id at block")
	}

	logger.Debugf("Applied authority set forced change: %s", forcedChange)

	s.forcedChanges.pruneAll()
	s.scheduledChangeRoots.pruneAll()
	return nil
}

// NextGrandpaAuthorityChange returns the block number of the next upcoming grandpa authorities change.
// It returns 0 if no change is scheduled.
func (s *GrandpaState) NextGrandpaAuthorityChange(bestBlockHash common.Hash, bestBlockNumber uint) (
	blockNumber uint, err error) {
	forcedChange, err := s.forcedChanges.lookupChangeWhere(func(pc pendingChange) (bool, error) {
		isDecendant, err := s.blockState.IsDescendantOf(pc.announcingHeader.Hash(), bestBlockHash)
		if err != nil {
			return false, fmt.Errorf("cannot check ancestry: %w", err)
		}

		return isDecendant && pc.effectiveNumber() <= bestBlockNumber, nil
	})
	if err != nil {
		return 0, fmt.Errorf("cannot get forced change on chain of %s: %w",
			bestBlockHash, err)
	}

	scheduledChangeNode, err := s.scheduledChangeRoots.lookupChangeWhere(func(pcn *pendingChangeNode) (bool, error) {
		isDecendant, err := s.blockState.IsDescendantOf(pcn.change.announcingHeader.Hash(), bestBlockHash)
		if err != nil {
			return false, fmt.Errorf("cannot check ancestry: %w", err)
		}

		return isDecendant && pcn.change.effectiveNumber() <= bestBlockNumber, nil
	})
	if err != nil {
		return 0, fmt.Errorf("cannot get forced change on chain of %s: %w",
			bestBlockHash, err)
	}

	var next uint
	if scheduledChangeNode != nil {
		next = scheduledChangeNode.change.effectiveNumber()
	}

	if forcedChange != nil && (forcedChange.effectiveNumber() < next || next == 0) {
		next = forcedChange.effectiveNumber()
	}

	if next == 0 {
		return 0, ErrNoNextAuthorityChange
	}

	return next, nil
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
	enc, err := types.EncodeGrandpaVoters(authorities)
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

	v, err := types.DecodeGrandpaVoters(enc)
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

// SetNextChange sets the next authority change at the given block number.
// NOTE: This block number will be the last block in the current set and not part of the next set.
func (s *GrandpaState) SetNextChange(authorities []types.GrandpaVoter, number uint) error {
	currSetID, err := s.GetCurrentSetID()
	if err != nil {
		return err
	}

	nextSetID := currSetID + 1
	err = s.setAuthorities(nextSetID, authorities)
	if err != nil {
		return err
	}

	err = s.setChangeSetIDAtBlock(nextSetID, number)
	if err != nil {
		return err
	}

	return nil
}

// IncrementSetID increments the set ID
func (s *GrandpaState) IncrementSetID() (newSetID uint64, err error) {
	currSetID, err := s.GetCurrentSetID()
	if err != nil {
		return 0, fmt.Errorf("cannot get current set ID: %w", err)
	}

	newSetID = currSetID + 1
	err = s.setCurrentSetID(newSetID)
	if err != nil {
		return 0, fmt.Errorf("cannot set current set ID: %w", err)
	}

	return newSetID, nil
}

// setSetIDChangeAtBlock sets a set ID change at a certain block
func (s *GrandpaState) setChangeSetIDAtBlock(setID uint64, number uint) error {
	return s.db.Put(setIDChangeKey(setID), common.UintToBytes(number))
}

// GetSetIDChange returns the block number where the set ID was updated
func (s *GrandpaState) GetSetIDChange(setID uint64) (blockNumber uint, err error) {
	num, err := s.db.Get(setIDChangeKey(setID))
	if err != nil {
		return 0, err
	}

	return common.BytesToUint(num), nil
}

// GetSetIDByBlockNumber returns the set ID for a given block number
func (s *GrandpaState) GetSetIDByBlockNumber(blockNumber uint) (uint64, error) {
	curr, err := s.GetCurrentSetID()
	if err != nil {
		return 0, err
	}

	for {
		changeUpper, err := s.GetSetIDChange(curr + 1)
		if errors.Is(err, pebble.ErrNotFound) {
			if curr == 0 {
				return 0, nil
			}
			curr = curr - 1
			continue
		} else if err != nil {
			return 0, err
		}

		changeLower, err := s.GetSetIDChange(curr)
		if err != nil {
			return 0, err
		}

		// Set id changes at the last block in the set. So, block (changeLower) at which current
		// set id was set, does not belong to current set. Thus, all block numbers in given set
		// would be more than changeLower.
		// Next set id change happens at the last block of current set. Thus, a block number from
		// given set could be lower or equal to changeUpper.
		if blockNumber <= changeUpper && blockNumber > changeLower {
			return curr, nil
		}

		if blockNumber > changeUpper {
			return curr + 1, nil
		}

		curr = curr - 1

		if int(curr) < 0 {
			return 0, nil
		}
	}
}

// SetNextPause sets the next grandpa pause at the given block number
func (s *GrandpaState) SetNextPause(number uint) error {
	value := common.UintToBytes(number)
	return s.db.Put(pauseKey, value)
}

// GetNextPause returns the block number of the next grandpa pause.
// If the key is not found in the database, the error pebble.ErrNotFound
// is returned.
func (s *GrandpaState) GetNextPause() (blockNumber uint, err error) {
	value, err := s.db.Get(pauseKey)
	if err != nil {
		return 0, err
	}

	return common.BytesToUint(value), nil
}

// SetNextResume sets the next grandpa resume at the given block number
func (s *GrandpaState) SetNextResume(number uint) error {
	value := common.UintToBytes(number)
	return s.db.Put(resumeKey, value)
}

// GetNextResume returns the block number of the next grandpa resume.
// If the key is not found in the database, the error pebble.ErrNotFound
// is returned.
func (s *GrandpaState) GetNextResume() (blockNumber uint, err error) {
	value, err := s.db.Get(resumeKey)
	if err != nil {
		return 0, err
	}

	return common.BytesToUint(value), nil
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
func (s *GrandpaState) SetPrevotes(round, setID uint64, pvs []types.GrandpaSignedVote) error {
	data, err := scale.Marshal(pvs)
	if err != nil {
		return err
	}

	return s.db.Put(prevotesKey(round, setID), data)
}

// GetPrevotes retrieves the prevotes for a specific round and set ID from the database
func (s *GrandpaState) GetPrevotes(round, setID uint64) ([]types.GrandpaSignedVote, error) {
	data, err := s.db.Get(prevotesKey(round, setID))
	if err != nil {
		return nil, err
	}

	pvs := []types.GrandpaSignedVote{}
	err = scale.Unmarshal(data, &pvs)
	if err != nil {
		return nil, err
	}

	return pvs, nil
}

// SetPrecommits sets the precommits for a specific round and set ID in the database
func (s *GrandpaState) SetPrecommits(round, setID uint64, pcs []types.GrandpaSignedVote) error {
	data, err := scale.Marshal(pcs)
	if err != nil {
		return err
	}

	return s.db.Put(precommitsKey(round, setID), data)
}

// GetPrecommits retrieves the precommits for a specific round and set ID from the database
func (s *GrandpaState) GetPrecommits(round, setID uint64) ([]types.GrandpaSignedVote, error) {
	data, err := s.db.Get(precommitsKey(round, setID))
	if err != nil {
		return nil, err
	}

	pcs := []types.GrandpaSignedVote{}
	err = scale.Unmarshal(data, &pcs)
	if err != nil {
		return nil, err
	}

	return pcs, nil
}
