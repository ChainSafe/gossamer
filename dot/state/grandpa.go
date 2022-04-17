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
	genesisSetID      = uint64(0)
	grandpaPrefix     = "grandpa"
	authoritiesPrefix = []byte("auth")
	setIDChangePrefix = []byte("change")
	pauseKey          = []byte("pause")
	resumeKey         = []byte("resume")
	currentSetIDKey   = []byte("setID")
)

var (
	ErrAlreadyHasForcedChanges = errors.New("already has a forced change")
	ErrUnfinalizedAncestor     = errors.New("ancestor with changes not applied")
	ErrNoChanges               = errors.New("cannot get the next authority change block number")
	ErrLowerThanBestFinalized  = errors.New("current finalized is lower than best finalized header")
)

type pendingChange struct {
	delay            uint32
	nextAuthorities  []types.Authority
	announcingHeader *types.Header
}

func (p pendingChange) String() string {
	return fmt.Sprintf("announcing header: %s (%d), delay: %d, next authorities: %d",
		p.announcingHeader.Hash(), p.announcingHeader.Number, p.delay, len(p.nextAuthorities))
}

func (p *pendingChange) effectiveNumber() uint {
	return p.announcingHeader.Number + uint(p.delay)
}

type isDescendantOfFunc func(parent, child common.Hash) (bool, error)

type pendingChangeNode struct {
	header *types.Header
	change *pendingChange
	nodes  []*pendingChangeNode
}

func (c *pendingChangeNode) importScheduledChange(header *types.Header, pendingChange *pendingChange,
	isDescendantOf isDescendantOfFunc) (imported bool, err error) {
	if c.header.Hash() == header.Hash() {
		return false, errors.New("duplicate block hash while importing change")
	}

	if header.Number <= c.header.Number {
		return false, nil
	}

	for _, childrenNodes := range c.nodes {
		imported, err := childrenNodes.importScheduledChange(header, pendingChange, isDescendantOf)
		if err != nil {
			return false, fmt.Errorf("could not import change: %w", err)
		}

		if imported {
			return true, nil
		}
	}

	isDescendant, err := isDescendantOf(c.header.Hash(), header.Hash())
	if err != nil {
		return false, fmt.Errorf("cannot define ancestry: %w", err)
	}

	if !isDescendant {
		return false, nil
	}

	pendingChangeNode := &pendingChangeNode{header: header, change: pendingChange, nodes: []*pendingChangeNode{}}
	c.nodes = append(c.nodes, pendingChangeNode)
	return true, nil
}

type orderedPendingChanges []*pendingChange

func (o orderedPendingChanges) Len() int      { return len(o) }
func (o orderedPendingChanges) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

// Less order by first effective number then by block number
func (o orderedPendingChanges) Less(i, j int) bool {
	return o[i].effectiveNumber() < o[j].effectiveNumber() &&
		o[i].announcingHeader.Number < o[j].announcingHeader.Number
}

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
		return nil, err
	}

	if err := s.SetLatestRound(0); err != nil {
		return nil, err
	}

	if err := s.setAuthorities(genesisSetID, genesisAuthorities); err != nil {
		return nil, err
	}

	if err := s.setChangeSetIDAtBlock(genesisSetID, 0); err != nil {
		return nil, err
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
		return nil
	case types.GrandpaResume:
		return nil
	default:
		return fmt.Errorf("not supported digest")
	}
}

func (s *GrandpaState) addForcedChange(header *types.Header, fc types.GrandpaForcedChange) error {
	headerHash := header.Hash()

	for _, change := range s.forcedChanges {
		changeBlockHash := change.announcingHeader.Hash()

		if changeBlockHash == headerHash {
			return errors.New("duplicated hash")
		}

		isDescendant, err := s.blockState.IsDescendantOf(changeBlockHash, headerHash)
		if err != nil {
			return fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if isDescendant {
			return errors.New("multiple forced changes")
		}
	}

	auths, err := types.GrandpaAuthoritiesRawToAuthorities(fc.Auths)
	if err != nil {
		return fmt.Errorf("cannot parser GRANPDA authorities to raw authorities: %w", err)
	}

	pendingChange := &pendingChange{
		nextAuthorities:  auths,
		announcingHeader: header,
		delay:            fc.Delay,
	}

	s.forcedChanges = append(s.forcedChanges, pendingChange)
	sort.Sort(s.forcedChanges)

	return nil
}

func (s *GrandpaState) addScheduledChange(header *types.Header, sc types.GrandpaScheduledChange) error {
	auths, err := types.GrandpaAuthoritiesRawToAuthorities(sc.Auths)
	if err != nil {
		return fmt.Errorf("cannot parser GRANPDA authorities to raw authorities: %w", err)
	}

	pendingChange := &pendingChange{
		nextAuthorities:  auths,
		announcingHeader: header,
		delay:            sc.Delay,
	}

	return s.importScheduledChange(header, pendingChange)
}

func (s *GrandpaState) importScheduledChange(header *types.Header, pendingChange *pendingChange) error {
	defer func() {
		logger.Debugf("there are now %d possible standard changes (roots)", len(s.scheduledChangeRoots))
	}()

	highestFinalizedHeader, err := s.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return fmt.Errorf("cannot get highest finalized header: %w", err)
	}

	if header.Number <= highestFinalizedHeader.Number {
		return errors.New("cannot import changes from blocks older then our highest finalized block")
	}

	for _, root := range s.scheduledChangeRoots {
		imported, err := root.importScheduledChange(header, pendingChange, s.blockState.IsDescendantOf)

		if err != nil {
			return fmt.Errorf("could not import change: %w", err)
		}

		if imported {
			logger.Debugf("changes on header %s (%d) imported succesfully", header.Hash(), header.Number)
			return nil
		}
	}

	pendingChangeNode := &pendingChangeNode{header: header, change: pendingChange, nodes: []*pendingChangeNode{}}
	s.scheduledChangeRoots = append(s.scheduledChangeRoots, pendingChangeNode)
	return nil
}

// getApplicableChange iterates throught the scheduled change tree roots looking for the change node, which
// contains a lower or equal effective number, to apply. When we found that node we update the scheduled change tree roots
// with its children that belongs to the same finalized node branch. If we don't find such node we update the scheduled
// change tree roots with the change nodes that belongs to the same finalized node branch
func (s *GrandpaState) getApplicableChange(finalizedHash common.Hash, finalizedNumber uint) (change *pendingChange, err error) {
	bestFinalizedHeader, err := s.blockState.GetHighestFinalisedHeader()
	if err != nil {
		return nil, fmt.Errorf("cannot get highest finalised header: %w", err)
	}

	if finalizedNumber < bestFinalizedHeader.Number {
		return nil, ErrLowerThanBestFinalized
	}

	effectiveBlockLowerOrEqualFinalized := func(change *pendingChange) bool {
		return change.effectiveNumber() <= finalizedNumber
	}

	position := -1
	for idx, root := range s.scheduledChangeRoots {
		if !effectiveBlockLowerOrEqualFinalized(root.change) {
			continue
		}

		rootHash := root.header.Hash()

		// if the current root doesn't have the same hash
		// neither the finalized header is descendant of the root then skip to the next root
		if !finalizedHash.Equal(rootHash) {
			isDescendant, err := s.blockState.IsDescendantOf(rootHash, finalizedHash)
			if err != nil {
				return nil, fmt.Errorf("cannot verify ancestry: %w", err)
			}

			if !isDescendant {
				continue
			}
		}

		// as the changes needs to be applied in order we need to check if our finalized
		// header is in front of any children, if it is that means some previous change was not applied
		for _, node := range root.nodes {
			isDescendant, err := s.blockState.IsDescendantOf(node.header.Hash(), finalizedHash)
			if err != nil {
				return nil, fmt.Errorf("cannot verify ancestry: %w", err)
			}

			if node.header.Number <= finalizedNumber && isDescendant {
				return nil, ErrUnfinalizedAncestor
			}
		}

		position = idx
		break
	}

	var changeToApply *pendingChange = nil

	if position > -1 {
		pendingChangeNodeAtPosition := s.scheduledChangeRoots[position]
		changeToApply = pendingChangeNodeAtPosition.change

		s.scheduledChangeRoots = make([]*pendingChangeNode, len(pendingChangeNodeAtPosition.nodes))
		copy(s.scheduledChangeRoots, pendingChangeNodeAtPosition.nodes)
	}

	return changeToApply, nil
}

// keepDescendantForcedChanges should keep the forced changes for later blocks that
// are descendant of the finalized block
func (s *GrandpaState) keepDescendantForcedChanges(finalizedHash common.Hash, finalizedNumber uint) error {
	onBranchForcedChanges := []*pendingChange{}

	for _, forcedChange := range s.forcedChanges {
		isDescendant, err := s.blockState.IsDescendantOf(finalizedHash, forcedChange.announcingHeader.Hash())
		if err != nil {
			return fmt.Errorf("cannot verify ancestry while ancestor: %w", err)
		}

		if forcedChange.effectiveNumber() > finalizedNumber && isDescendant {
			onBranchForcedChanges = append(onBranchForcedChanges, forcedChange)
		}
	}

	s.forcedChanges = make(orderedPendingChanges, len(onBranchForcedChanges))
	copy(s.forcedChanges, onBranchForcedChanges)

	return nil
}

// ApplyScheduledChange will check the schedules changes in order to find a root
// that is equals or behind the finalized number and will apply its authority set changes
func (s *GrandpaState) ApplyScheduledChanges(finalizedHeader *types.Header) error {
	finalizedHash := finalizedHeader.Hash()

	err := s.keepDescendantForcedChanges(finalizedHash, finalizedHeader.Number)
	if err != nil {
		return fmt.Errorf("cannot keep descendant forced changes: %w", err)
	}

	if len(s.scheduledChangeRoots) == 0 {
		return nil
	}

	changeToApply, err := s.getApplicableChange(finalizedHash, finalizedHeader.Number)
	if err != nil {
		return fmt.Errorf("cannot finalize scheduled change: %w", err)
	}

	logger.Debugf("scheduled changes: change to apply: %s", changeToApply)

	if changeToApply != nil {
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
			return fmt.Errorf("cannot set change set id at block")
		}

		logger.Debugf("Applying authority set change scheduled at block #%d",
			changeToApply.announcingHeader.Number)

		// TODO: add afg.applying_scheduled_authority_set_change telemetry info here
	}

	return nil
}

// forcedChangeOnChain walk through the forced change slice looking for
// a forced change that belong to the same branch as bestBlockHash parameter
func (s *GrandpaState) forcedChangeOnChain(bestBlockHash common.Hash) (change *pendingChange, err error) {
	for _, forcedChange := range s.forcedChanges {
		forcedChangeHeader := forcedChange.announcingHeader

		var isDescendant bool
		isDescendant, err = s.blockState.IsDescendantOf(
			forcedChangeHeader.Hash(), bestBlockHash)

		if err != nil {
			return nil, fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if !isDescendant {
			continue
		}

		return forcedChange, nil
	}

	return nil, nil
}

// scheduledChangeOnChainOf walk only through the scheduled changes roots slice looking for
// a scheduled change that belong to the same branch as bestBlockHash parameter
func (s *GrandpaState) scheduledChangeOnChainOf(bestBlockHash common.Hash) (change *pendingChange, err error) {
	for _, scheduledChange := range s.scheduledChangeRoots {
		var isDescendant bool
		isDescendant, err = s.blockState.IsDescendantOf(
			scheduledChange.header.Hash(), bestBlockHash)

		if err != nil {
			return nil, fmt.Errorf("cannot verify ancestry: %w", err)
		}

		if !isDescendant {
			continue
		}

		return scheduledChange.change, nil
	}

	return nil, nil
}

func (s *GrandpaState) ApplyForcedChanges(bestBlockHeader *types.Header) error {
	return nil
}

// NextGrandpaAuthorityChange returns the block number of the next upcoming grandpa authorities change.
// It returns 0 if no change is scheduled.
func (s *GrandpaState) NextGrandpaAuthorityChange(bestBlockHash common.Hash) (blockNumber uint, err error) {
	forcedChange, err := s.forcedChangeOnChain(bestBlockHash)
	if err != nil {
		return 0, fmt.Errorf("cannot get forced change: %w", err)
	}

	scheduledChange, err := s.scheduledChangeOnChainOf(bestBlockHash)
	if err != nil {
		return 0, fmt.Errorf("cannot get scheduled change: %w", err)
	}

	if forcedChange == nil && scheduledChange == nil {
		return 0, ErrNoChanges
	}

	if forcedChange.announcingHeader.Number < scheduledChange.announcingHeader.Number {
		return forcedChange.effectiveNumber(), nil
	}

	return scheduledChange.effectiveNumber(), nil
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

// SetNextChange sets the next authority change
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

// GetSetIDChange returs the block number where the set ID was updated
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

		// if the given block number is in the range of changeLower < blockNumber <= changeUpper
		// return the set id to the change lower
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
