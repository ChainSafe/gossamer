// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package babe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/telemetry"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"

	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "babe"))

// Service contains the VRF keys for the validator, as well as BABE configuation data
type Service struct {
	ctx          context.Context
	cancel       context.CancelFunc
	authority    bool
	dev          bool
	constants    constants
	epochHandler *epochHandler

	// Storage interfaces
	blockState       BlockState
	storageState     StorageState
	transactionState TransactionState
	epochState       EpochState

	blockImportHandler BlockImportHandler

	// BABE authority keypair
	keypair *sr25519.Keypair // TODO: change to BABE keystore (#1864)

	// State variables
	sync.RWMutex
	pause chan struct{}

	telemetry Telemetry
	wg        sync.WaitGroup
}

// ServiceConfig represents a BABE configuration
type ServiceConfig struct {
	LogLvl             log.Level
	BlockState         BlockState
	StorageState       StorageState
	TransactionState   TransactionState
	EpochState         EpochState
	BlockImportHandler BlockImportHandler
	Keypair            *sr25519.Keypair
	AuthData           []types.Authority
	IsDev              bool
	Authority          bool
	Telemetry          Telemetry
}

// Validate returns error if config does not contain required attributes
func (sc *ServiceConfig) Validate() error {
	if sc.Keypair == nil && sc.Authority {
		return errNoBABEAuthorityKeyProvided
	}

	return nil
}

// Builder struct to hold babe builder functions
type Builder struct{}

// NewServiceIFace returns a new Babe Service using the provided VRF keys and runtime
func (Builder) NewServiceIFace(cfg *ServiceConfig) (service *Service, err error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("could not verify service config: %w", err)
	}

	logger.Patch(log.SetLevel(cfg.LogLvl))

	slotDuration, err := cfg.EpochState.GetSlotDuration()
	if err != nil {
		return nil, fmt.Errorf("cannot get slot duration: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	babeService := &Service{
		ctx:                ctx,
		cancel:             cancel,
		blockState:         cfg.BlockState,
		storageState:       cfg.StorageState,
		epochState:         cfg.EpochState,
		keypair:            cfg.Keypair,
		transactionState:   cfg.TransactionState,
		pause:              make(chan struct{}),
		authority:          cfg.Authority,
		dev:                cfg.IsDev,
		blockImportHandler: cfg.BlockImportHandler,
		constants: constants{
			slotDuration: slotDuration,
			epochLength:  cfg.EpochState.GetEpochLength(),
		},
		telemetry: cfg.Telemetry,
	}

	logger.Debugf(
		"created service with block producer ID=%v, slot duration %s, epoch length (slots) %d",
		cfg.Authority, babeService.constants.slotDuration, babeService.constants.epochLength,
	)

	return babeService, nil
}

// NewService function to create babe service
func NewService(cfg *ServiceConfig) (*Service, error) {
	if cfg.Keypair == nil && cfg.Authority {
		return nil, errors.New("cannot create BABE service as authority; no keypair provided")
	}

	logger.Patch(log.SetLevel(cfg.LogLvl))

	slotDuration, err := cfg.EpochState.GetSlotDuration()
	if err != nil {
		return nil, fmt.Errorf("cannot get slot duration: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	babeService := &Service{
		ctx:                ctx,
		cancel:             cancel,
		blockState:         cfg.BlockState,
		storageState:       cfg.StorageState,
		epochState:         cfg.EpochState,
		keypair:            cfg.Keypair,
		transactionState:   cfg.TransactionState,
		pause:              make(chan struct{}),
		authority:          cfg.Authority,
		dev:                cfg.IsDev,
		blockImportHandler: cfg.BlockImportHandler,
		constants: constants{
			slotDuration: slotDuration,
			epochLength:  cfg.EpochState.GetEpochLength(),
		},
		telemetry: cfg.Telemetry,
	}

	logger.Debugf(
		"created service with block producer ID=%v, slot duration %s, epoch length (slots) %d",
		cfg.Authority, babeService.constants.slotDuration, babeService.constants.epochLength,
	)

	return babeService, nil
}

// Start starts BABE block authoring
func (b *Service) Start() error {
	if !b.authority {
		return nil
	}

	b.wg.Add(1)
	go func() {
		b.initiate()
		b.wg.Done()
	}()
	return nil
}

// SlotDuration returns the current service slot duration in milliseconds
func (b *Service) SlotDuration() uint64 {
	return uint64(b.constants.slotDuration.Milliseconds())
}

// EpochLength returns the current service epoch duration
func (b *Service) EpochLength() uint64 {
	return b.constants.epochLength
}

// Pause pauses the service ie. halts block production
func (b *Service) Pause() error {
	b.Lock()
	defer b.Unlock()

	if b.IsPaused() {
		return nil
	}

	close(b.pause)
	return nil
}

// Resume resumes the service ie. resumes block production
func (b *Service) Resume() error {
	b.Lock()
	defer b.Unlock()

	if !b.IsPaused() {
		return nil
	}

	b.pause = make(chan struct{})
	b.wg.Add(1)
	go func() {
		b.initiate()
		b.wg.Done()
	}()
	logger.Debug("service resumed")
	return nil
}

// IsPaused returns if the service is paused or not (ie. producing blocks)
func (b *Service) IsPaused() bool {
	select {
	case <-b.pause:
		return true
	default:
		return false
	}
}

// Stop stops the service. If stop is called, it cannot be resumed.
func (b *Service) Stop() error {
	if !b.authority {
		return nil
	}

	b.Lock()
	defer b.Unlock()

	if b.ctx.Err() != nil {
		return errors.New("service already stopped")
	}

	ethmetrics.Unregister(buildBlockTimer)
	ethmetrics.Unregister(buildBlockErrors)

	b.cancel()
	b.wg.Wait()
	return nil
}

// AuthoritiesRaw returns the current BABE authorities
func (b *Service) AuthoritiesRaw() []types.AuthorityRaw {
	return b.epochHandler.descriptor.data.authorities
}

// IsStopped returns true if the service is stopped (ie not producing blocks)
func (b *Service) IsStopped() bool {
	return b.ctx.Err() != nil
}

func (b *Service) getAuthorityIndex(authorities []types.AuthorityRaw) (uint32, error) {
	if !b.authority {
		return 0, ErrNotAuthority
	}

	pub := b.keypair.Public()

	for i, auth := range authorities {
		if bytes.Equal(pub.Encode(), auth.Key[:]) {
			return uint32(i), nil
		}
	}

	return 0, fmt.Errorf("key not in BABE authority data")
}

func (b *Service) initiate() {
	// we should consider better error handling for this - we should
	// retry to run the engine at some point (maybe the next epoch) if
	// there's an error.
	if err := b.runEngine(); err != nil {
		logger.Criticalf("failed to run block production engine: %s", err)
	}
}

func (b *Service) initiateAndGetEpochHandler(epoch uint64) (*epochHandler, error) {
	epochDescriptor, err := b.initiateEpoch(epoch)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate epoch: %w", err)
	}

	logger.Debugf("initiated epoch with threshold %s, randomness 0x%x and authorities %v",
		epochDescriptor.data.threshold, epochDescriptor.data.randomness[:], epochDescriptor.data.authorities)

	return newEpochHandler(
		epochDescriptor,
		b.constants,
		b.handleSlot,
		b.keypair,
	)
}

func (b *Service) runEngine() error {
	epoch, err := b.epochState.GetCurrentEpoch()
	if err != nil {
		return fmt.Errorf("failed to get current epoch: %s", err)
	}

	for {
		next, err := b.handleEpoch(epoch)
		if errors.Is(err, errServicePaused) || errors.Is(err, context.Canceled) {
			return nil
		} else if err != nil {
			return fmt.Errorf("cannot handle epoch: %w", err)
		}

		epoch = next
	}
}

func (b *Service) handleEpoch(epoch uint64) (next uint64, err error) {
	wg := sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		wg.Wait()
	}()

	b.epochHandler, err = b.initiateAndGetEpochHandler(epoch)
	if err != nil {
		return 0, fmt.Errorf("cannot initiate and get epoch handler: %w", err)
	}

	nextEpochStarts := b.epochHandler.descriptor.endSlot
	nextEpochStartTime := getSlotStartTime(nextEpochStarts, b.constants.slotDuration)
	epochTimer := time.NewTimer(time.Until(nextEpochStartTime))

	errCh := make(chan error, 1)
	wg.Add(1)
	go func() {
		b.epochHandler.run(ctx, errCh)
		wg.Done()
	}()

	select {
	case <-b.ctx.Done():
		epochTimer.Stop()
		return 0, b.ctx.Err()
	case <-b.pause:
		epochTimer.Stop()
		return 0, errServicePaused
	case <-epochTimer.C:
		// stop current epoch handler
		cancel()
	case err := <-errCh:
		// TODO: errEpochPast is sent on this channel, but it doesnot get logged here
		epochTimer.Stop()
		if err != nil {
			logger.Errorf("error from epochHandler: %s", err)
		}
	}

	// setup next epoch, re-invoke block authoring
	next, err = b.incrementEpoch()
	if err != nil {
		return 0, fmt.Errorf("failed to increment epoch: %w", err)
	}

	logger.Infof("epoch %d complete, upcoming epoch: %d", epoch, next)
	return next, nil
}

func (b *Service) getParentForBlockAuthoring(slotNum uint64) (*types.Header, error) {
	parentHeader, err := b.blockState.BestBlockHeader()
	if err != nil {
		return nil, fmt.Errorf("could not get best block header: %w", err)
	}

	if parentHeader == nil {
		return nil, errNilParentHeader
	}

	atGenesisBlock := b.blockState.GenesisHash() == parentHeader.Hash()
	if !atGenesisBlock {
		bestBlockSlotNum, err := b.blockState.GetSlotForBlock(parentHeader.Hash())
		if err != nil {
			return nil, fmt.Errorf("could not get slot for block: %w", err)
		}

		if bestBlockSlotNum > slotNum {
			return nil, fmt.Errorf("%w: best block slot number is %d and got slot number %d",
				errLaggingSlot, bestBlockSlotNum, slotNum)
		}

		if bestBlockSlotNum == slotNum {
			// pick parent of best block instead to handle slot
			newParentHeader, err := b.blockState.GetHeader(parentHeader.ParentHash)
			if err != nil {
				return nil, fmt.Errorf("could not get header: %w", err)
			}
			if newParentHeader == nil {
				return nil, fmt.Errorf("%w: for block hash %s", errNilParentHeader, parentHeader.ParentHash)
			}
			parentHeader = newParentHeader
		}
	}

	// there is a chance that the best block header may change in the course of building the block,
	// so let's copy it first.
	parent, err := parentHeader.DeepCopy()
	if err != nil {
		return nil, fmt.Errorf("could not create deep copy of parent header: %w", err)
	}

	return parent, nil
}

func (b *Service) handleSlot(epoch uint64, slot Slot,
	authorityIndex uint32,
	preRuntimeDigest *types.PreRuntimeDigest,
) error {
	parent, err := b.getParentForBlockAuthoring(slot.number)
	if err != nil {
		return fmt.Errorf("could not get parent for claiming slot %d: %w", slot.number, err)
	}
	b.storageState.Lock()
	defer b.storageState.Unlock()

	// set runtime trie before building block
	// if block building is successful, store the resulting trie in the storage state
	ts, err := b.storageState.TrieState(&parent.StateRoot)
	if err != nil || ts == nil {
		logger.Errorf("failed to get parent trie with parent state root %s: %s", parent.StateRoot, err)
		return err
	}

	rt, err := b.blockState.GetRuntime(parent.Hash())
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)

	block, err := b.buildBlock(parent, slot, rt, authorityIndex, preRuntimeDigest)
	if err != nil {
		return err
	}

	logger.Infof(
		"built block %d with hash %s, state root %s, epoch %d and slot %d",
		block.Header.Number, block.Header.Hash(), block.Header.StateRoot, epoch, slot.number)
	logger.Tracef(
		"built block with parent hash %s, header %s and body %s",
		parent.Hash(), block.Header.String(), block.Body)

	b.telemetry.SendMessage(
		telemetry.NewPreparedBlockForProposing(
			block.Header.Hash(),
			fmt.Sprint(block.Header.Number),
		),
	)

	if err := b.blockImportHandler.HandleBlockProduced(block, ts); err != nil {
		logger.Warnf("failed to import built block: %s", err)
		return err
	}

	return nil
}

func getCurrentSlot(slotDuration time.Duration) uint64 {
	return uint64(time.Now().UnixNano()) / uint64(slotDuration.Nanoseconds())
}

func getSlotStartTime(slot uint64, slotDuration time.Duration) time.Time {
	return time.Unix(0, int64(slot)*slotDuration.Nanoseconds())
}
