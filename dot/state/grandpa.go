// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	errPendingScheduledChanges = errors.New("pending scheduled changes needs to be applied")
	errDuplicateHashes         = errors.New("duplicated hashes")
	errAlreadyHasForcedChanges = errors.New("already has a forced change")
	errUnfinalizedAncestor     = errors.New("ancestor with changes not applied")
	errLowerThanBestFinalized  = errors.New("current finalized is lower than best finalized header")

	ErrNoChanges = errors.New("cannot get the next authority change block number")
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
	db         chaindb.Database
	blockState *BlockState

	forksLock sync.RWMutex

	scheduledChangeRoots []*pendingChangeNode
	forcedChanges        orderedPendingChanges
}

// NewGrandpaStateFromGenesis returns a new GrandpaState given the grandpa genesis authorities
func NewGrandpaStateFromGenesis(db chaindb.Database, bs *BlockState,
	genesisAuthorities []types.GrandpaVoter) (*GrandpaState, error) {
	grandpaDB := chaindb.NewTable(db, grandpaPrefix)
	s := &GrandpaState{
		db:         grandpaDB,
		blockState: bs,
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
func NewGrandpaState(db chaindb.Database, bs *BlockState) (*GrandpaState, error) {
	return &GrandpaState{
		db:         chaindb.NewTable(db, grandpaPrefix),
		blockState: bs,
	}, nil
}

// HandleGRANDPADigest receives a decoded GRANDPA digest and calls the right function to handles the digest
func (s *GrandpaState) HandleGRANDPADigest(header *types.Header, digest scale.VaryingDataType) error {
	switch val := digest.Value().(type) {
	case types.GrandpaScheduledChange:
		return s.addScheduledChange(header, val)
	case types.GrandpaForcedChange:
		return s.addForcedChange(header, val)
	case types.GrandpaOnDisabled:
		return nil
	case types.GrandpaPause:
		logger.Warn("GRANDPA Pause consensus message not imeplemented yet")
		return nil
	case types.GrandpaResume:
		logger.Warn("GRANDPA Resume consensus message not imeplemented yet")
		return nil
	default:
		return fmt.Errorf("not supported digest")
	}
}

func (s *GrandpaState) addForcedChange(header *types.Header, fc types.GrandpaForcedChange) error {
	headerHash := header.Hash()

	for _, change := range s.forcedChanges {
		changeBlockHash := change.announcingHeader.Hash()

		if changeBlockHash.Equal(headerHash) {
			return errDuplicateHashes
		}

		isDescendant, err := s.blockState.IsDescendantOf(changeBlockHash, headerHash)
		if err != nil {
			return fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if isDescendant {
			return errAlreadyHasForcedChanges
		}
	}

	auths, err := types.GrandpaAuthoritiesRawToAuthorities(fc.Auths)
	if err != nil {
		return fmt.Errorf("cannot parser GRANDPA authorities to raw authorities: %w", err)
	}

	pendingChange := &pendingChange{
		bestFinalizedNumber: fc.BestFinalizedBlock,
		nextAuthorities:     auths,
		announcingHeader:    header,
		delay:               fc.Delay,
	}

	s.forcedChanges = append(s.forcedChanges, pendingChange)
	sort.Sort(s.forcedChanges)

	logger.Debugf("there are now %d possible forced changes", len(s.forcedChanges))
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

	return s.importScheduledChange(pendingChange)
}

func (s *GrandpaState) importScheduledChange(pendingChange *pendingChange) error {
	for _, root := range s.scheduledChangeRoots {
		imported, err := root.importScheduledChange(pendingChange.announcingHeader.Hash(),
			pendingChange.announcingHeader.Number, pendingChange, s.blockState.IsDescendantOf)

		if err != nil {
			return fmt.Errorf("could not import scheduled change: %w", err)
		}

		if imported {
			logger.Debugf("changes on header %s (%d) imported successfully",
				pendingChange.announcingHeader.Hash(), pendingChange.announcingHeader.Number)
			return nil
		}
	}

	pendingChangeNode := &pendingChangeNode{
		change: pendingChange,
		nodes:  []*pendingChangeNode{},
	}

	s.scheduledChangeRoots = append(s.scheduledChangeRoots, pendingChangeNode)
	logger.Debugf("there are now %d possible scheduled changes (roots)", len(s.scheduledChangeRoots))

	return nil
}

// getApplicableScheduledChange iterates through the scheduled change tree roots looking
// for the change node which contains a lower or equal effective number, to apply.
// When we found that node we update the scheduled change tree roots with its children
// that belongs to the same finalized node branch. If we don't find such node we update the scheduled
// change tree roots with the change nodes that belongs to the same finalized node branch
func (s *GrandpaState) getApplicableScheduledChange(finalizedHash common.Hash, finalizedNumber uint) (
	change *pendingChange, err error) {

	var changeNode *pendingChangeNode
	for _, root := range s.scheduledChangeRoots {
		if root.change.effectiveNumber() > finalizedNumber {
			continue
		}

		changeNodeHash := root.change.announcingHeader.Hash()

		// if the change doesn't have the same hash
		// neither the finalized header is descendant of the change then skip to the next root
		if !finalizedHash.Equal(changeNodeHash) {
			isDescendant, err := s.blockState.IsDescendantOf(changeNodeHash, finalizedHash)
			if err != nil {
				return nil, fmt.Errorf("cannot verify ancestry: %w", err)
			}

			if !isDescendant {
				continue
			}
		}

		// the changes must be applied in order, so we need to check if our finalized header
		// is ahead of any children, if it is that means some previous change was not applied
		for _, child := range root.nodes {
			isDescendant, err := s.blockState.IsDescendantOf(child.change.announcingHeader.Hash(), finalizedHash)
			if err != nil {
				return nil, fmt.Errorf("cannot verify ancestry: %w", err)
			}

			if child.change.announcingHeader.Number <= finalizedNumber && isDescendant {
				return nil, errUnfinalizedAncestor
			}
		}

		changeNode = root
		break
	}

	// if there is no change to be applied then we should keep only
	// the scheduled changes which belongs to the finalized header
	// otherwise we should update the scheduled roots to be the child
	// nodes of the applied scheduled change
	if changeNode == nil {
		err := s.pruneScheduledChanges(finalizedHash)
		if err != nil {
			return nil, fmt.Errorf("cannot prune non-descendant scheduled nodes: %w", err)
		}
	} else {
		change = changeNode.change
		s.scheduledChangeRoots = make([]*pendingChangeNode, len(changeNode.nodes))
		copy(s.scheduledChangeRoots, changeNode.nodes)
	}

	return change, nil
}

// pruneForcedChanges removes the forced changes from GRANDPA state for any block
// that is not a descendant of the current finalised block.
func (s *GrandpaState) pruneForcedChanges(finalizedHash common.Hash) error {
	onBranchForcedChanges := make([]*pendingChange, 0, len(s.forcedChanges))

	for _, forcedChange := range s.forcedChanges {
		isDescendant, err := s.blockState.IsDescendantOf(finalizedHash, forcedChange.announcingHeader.Hash())
		if err != nil {
			return fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if isDescendant {
			onBranchForcedChanges = append(onBranchForcedChanges, forcedChange)
		}
	}

	s.forcedChanges = make(orderedPendingChanges, len(onBranchForcedChanges))
	copy(s.forcedChanges, onBranchForcedChanges)

	return nil
}

// pruneScheduledChanges removes the scheduled changes from the
// GRANDPA state which are not for a descendant of the finalised block hash.
func (s *GrandpaState) pruneScheduledChanges(finalizedHash common.Hash) error {
	onBranchScheduledChanges := make([]*pendingChangeNode, 0, len(s.scheduledChangeRoots))

	for _, scheduledChange := range s.scheduledChangeRoots {
		scheduledChangeHash := scheduledChange.change.announcingHeader.Hash()

		isDescendant, err := s.blockState.IsDescendantOf(finalizedHash, scheduledChangeHash)
		if err != nil {
			return fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if isDescendant {
			onBranchScheduledChanges = append(onBranchScheduledChanges, scheduledChange)
		}
	}

	s.scheduledChangeRoots = make([]*pendingChangeNode, len(onBranchScheduledChanges))
	copy(s.scheduledChangeRoots, onBranchScheduledChanges)
	return nil
}

// ApplyScheduledChanges will check the schedules changes in order to find a root
// equal or behind the finalized number and will apply its authority set changes
func (s *GrandpaState) ApplyScheduledChanges(finalizedHeader *types.Header) error {
	finalizedHash := finalizedHeader.Hash()

	err := s.pruneForcedChanges(finalizedHash)
	if err != nil {
		return fmt.Errorf("cannot prune non-descendant forced changes: %w", err)
	}

	if len(s.scheduledChangeRoots) == 0 {
		return nil
	}

	changeToApply, err := s.getApplicableScheduledChange(finalizedHash, finalizedHeader.Number)
	if err != nil {
		return fmt.Errorf("cannot get applicable scheduled change: %w", err)
	}

	if changeToApply == nil {
		return nil
	}

	logger.Debugf("applying scheduled change: %s", changeToApply)

	newSetID, err := s.IncrementSetID()
	if err != nil {
		return fmt.Errorf("cannot increment set id: %w", err)
	}

	grandpaVotersAuthorities := types.NewGrandpaVotersFromAuthorities(changeToApply.nextAuthorities)
	err = s.setAuthorities(newSetID, grandpaVotersAuthorities)
	if err != nil {
		return fmt.Errorf("cannot set authorities: %w", err)
	}

	err = s.setChangeSetIDAtBlock(newSetID, changeToApply.effectiveNumber())
	if err != nil {
		return fmt.Errorf("cannot set the change set id at block: %w", err)
	}

	logger.Debugf("Applying authority set change scheduled at block #%d",
		changeToApply.announcingHeader.Number)

	// TODO: add afg.applying_scheduled_authority_set_change telemetry info here
	return nil
}

// ApplyForcedChanges will check for if there is a scheduled forced change relative to the
// imported block and then apply it otherwise nothing happens
func (s *GrandpaState) ApplyForcedChanges(importedBlockHeader *types.Header) error {
	importedBlockHash := importedBlockHeader.Hash()
	var forcedChange *pendingChange

	for _, forced := range s.forcedChanges {
		announcingHash := forced.announcingHeader.Hash()
		effectiveNumber := forced.effectiveNumber()

		if importedBlockHash.Equal(announcingHash) && effectiveNumber == importedBlockHeader.Number {
			forcedChange = forced
			break
		}

		isDescendant, err := s.blockState.IsDescendantOf(announcingHash, importedBlockHash)
		if err != nil {
			return fmt.Errorf("cannot check ancestry: %w", err)
		}

		if !isDescendant {
			continue
		}

		if effectiveNumber == importedBlockHeader.Number {
			forcedChange = forced
			break
		}
	}

	if forcedChange == nil {
		return nil
	}

	forcedChangeHash := forcedChange.announcingHeader.Hash()
	bestFinalizedNumber := forcedChange.bestFinalizedNumber

	// checking for dependant pending scheduled changes
	for _, scheduled := range s.scheduledChangeRoots {
		if scheduled.change.effectiveNumber() > uint(bestFinalizedNumber) {
			continue
		}

		scheduledBlockHash := scheduled.change.announcingHeader.Hash()
		isDescendant, err := s.blockState.IsDescendantOf(scheduledBlockHash, forcedChangeHash)
		if err != nil {
			return fmt.Errorf("cannot check ancestry: %w", err)
		}

		if isDescendant {
			return errPendingScheduledChanges
		}
	}

	logger.Debugf("applying forced change: %s", forcedChange)

	// send the telemetry s messages here
	// afg.applying_forced_authority_set_change

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

	logger.Debugf("Applying authority set forced change at block #%d",
		forcedChange.announcingHeader.Number)

	return nil
}

// forcedChangeOnChainOf walk through the forced change slice looking for
// a forced change that belong to the same branch as blockHash parameter
func (s *GrandpaState) forcedChangeOnChainOf(blockHash common.Hash) (forcedChange *pendingChange, err error) {
	for _, change := range s.forcedChanges {
		changeHeader := change.announcingHeader

		var isDescendant bool
		isDescendant, err = s.blockState.IsDescendantOf(
			changeHeader.Hash(), blockHash)

		if err != nil {
			return nil, fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if !isDescendant {
			continue
		}

		return change, nil
	}

	return forcedChange, nil
}

// scheduledChangeOnChainOf walk only through the scheduled changes roots slice looking for
// a scheduled change that belongs to the same branch as blockHash parameter
func (s *GrandpaState) scheduledChangeOnChainOf(blockHash common.Hash) (scheduledChange *pendingChange, err error) {
	for _, change := range s.scheduledChangeRoots {
		isDescendant, err := s.blockState.IsDescendantOf(
			change.change.announcingHeader.Hash(), blockHash)

		if err != nil {
			return nil, fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if !isDescendant {
			continue
		}

		return change.change, nil
	}

	return scheduledChange, nil
}

// NextGrandpaAuthorityChange returns the block number of the next upcoming grandpa authorities change.
// It returns 0 if no change is scheduled.
func (s *GrandpaState) NextGrandpaAuthorityChange(bestBlockHash common.Hash) (blockNumber uint, err error) {
	forcedChange, err := s.forcedChangeOnChainOf(bestBlockHash)
	if err != nil {
		return 0, fmt.Errorf("cannot get forced change on chain of %s: %w",
			bestBlockHash, err)
	}

	scheduledChange, err := s.scheduledChangeOnChainOf(bestBlockHash)
	if err != nil {
		return 0, fmt.Errorf("cannot get scheduled change on chain of %s: %w",
			bestBlockHash, err)
	}

	var next uint

	if scheduledChange != nil {
		next = scheduledChange.effectiveNumber()
	}

	if forcedChange != nil && forcedChange.effectiveNumber() < next {
		next = forcedChange.effectiveNumber()
	}

	if next == 0 {
		return 0, ErrNoChanges
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
		if errors.Is(err, chaindb.ErrKeyNotFound) {
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
// If the key is not found in the database, the error chaindb.ErrKeyNotFound
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
// If the key is not found in the database, the error chaindb.ErrKeyNotFound
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
