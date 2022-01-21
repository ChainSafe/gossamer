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
	ctx       context.Context
	cancel    context.CancelFunc
	authority bool
	dev       bool
	// lead is used when setting up a new network from genesis.
	// the "lead" node is the node that is designated to build block 1, after which the rest of the nodes
	// will sync block 1 and determine the first slot of the network based on it
	lead         bool
	constants    *constants
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

	telemetry telemetry.Client
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
	Lead               bool
	Telemetry          telemetry.Client
}

// NewService returns a new Babe Service using the provided VRF keys and runtime
func NewService(cfg *ServiceConfig) (*Service, error) {
	if cfg.Keypair == nil && cfg.Authority {
		return nil, errors.New("cannot create BABE service as authority; no keypair provided")
	}

	if cfg.BlockState == nil {
		return nil, errNilBlockState
	}

	if cfg.EpochState == nil {
		return nil, errNilEpochState
	}

	if cfg.BlockImportHandler == nil {
		return nil, errNilBlockImportHandler
	}

	logger.Patch(log.SetLevel(cfg.LogLvl))

	slotDuration, err := cfg.EpochState.GetSlotDuration()
	if err != nil {
		return nil, fmt.Errorf("cannot get slot duration: %w", err)
	}

	epochLength, err := cfg.EpochState.GetEpochLength()
	if err != nil {
		return nil, fmt.Errorf("cannot get epoch length: %w", err)
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
		lead:               cfg.Lead,
		constants: &constants{
			slotDuration: slotDuration,
			epochLength:  epochLength,
		},
		telemetry: cfg.Telemetry,
	}

	logger.Debugf(
		"created service with block producer ID=%v, slot duration %s, epoch length (slots) %d",
		cfg.Authority, babeService.constants.slotDuration, babeService.constants.epochLength,
	)

	if cfg.Lead {
		logger.Debug("node designated to build block 1")
	}

	return babeService, nil
}

// Start starts BABE block authoring
func (b *Service) Start() error {
	if !b.authority {
		return nil
	}

	// if we aren't leading node, wait for first block
	if !b.lead {
		if err := b.waitForFirstBlock(); err != nil {
			return err
		}
	}

	go b.initiate()
	return nil
}

func (b *Service) waitForFirstBlock() error {
	head, err := b.blockState.BestBlockHeader()
	if err != nil {
		return fmt.Errorf("cannot get best block header: %w", err)
	}

	if head.Number.Uint64() > 0 {
		return nil
	}

	ch := b.blockState.GetImportedBlockNotifierChannel()
	defer b.blockState.FreeImportedBlockNotifierChannel(ch)

	const firstBlockTimeout = time.Minute * 5
	timer := time.NewTimer(firstBlockTimeout)
	cleanup := func() {
		if !timer.Stop() {
			<-timer.C
		}
	}

	// loop until block 1
	for {
		select {
		case block, ok := <-ch:
			if !ok {
				cleanup()
				return errChannelClosed
			}

			if ok && block.Header.Number.Int64() > 0 {
				cleanup()
				return nil
			}
		case <-timer.C:
			return errFirstBlockTimeout
		case <-b.ctx.Done():
			cleanup()
			return b.ctx.Err()
		}
	}
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
	go b.initiate()
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
	return nil
}

// Authorities returns the current BABE authorities
func (b *Service) Authorities() []types.Authority {
	auths := make([]types.Authority, len(b.epochHandler.epochData.authorities))
	copy(auths, b.epochHandler.epochData.authorities)
	return auths
}

// IsStopped returns true if the service is stopped (ie not producing blocks)
func (b *Service) IsStopped() bool {
	return b.ctx.Err() != nil
}

func (b *Service) getAuthorityIndex(Authorities []types.Authority) (uint32, error) {
	if !b.authority {
		return 0, ErrNotAuthority
	}

	pub := b.keypair.Public()

	for i, auth := range Authorities {
		if bytes.Equal(pub.Encode(), auth.Key.Encode()) {
			return uint32(i), nil
		}
	}

	return 0, fmt.Errorf("key not in BABE authority data")
}

func (b *Service) getSlotDuration() time.Duration {
	return b.constants.slotDuration
}

func (b *Service) initiate() {
	if b.blockState == nil {
		logger.Errorf("block authoring: %s", errNilBlockState)
		return
	}

	if b.storageState == nil {
		logger.Errorf("block authoring: %s", errNilStorageState)
		return
	}

	// we should consider better error handling for this - we should
	// retry to run the engine at some point (maybe the next epoch) if
	// there's an error.
	if err := b.runEngine(); err != nil {
		logger.Criticalf("failed to run block production engine: %s", err)
	}
}

func (b *Service) initiateAndGetEpochHandler(ctx context.Context, epoch uint64) (*epochHandler, error) {
	epochData, err := b.initiateEpoch(epoch)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate epoch %d: %s", epoch, err)
	}

	logger.Debugf("initiated epoch with threshold %s, randomness 0x%x and authorities %v",
		epochData.threshold, epochData.randomness[:], epochData.authorities)

	epochStartSlot, err := b.epochState.GetStartSlotForEpoch(epoch)
	if err != nil {
		return nil, fmt.Errorf("failed to get start slot for current epoch %d: %s", epoch, err)
	}

	return newEpochHandler(ctx,
		epoch,
		epochStartSlot,
		epochData,
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
		err = func() error {
			ctx, cancel := context.WithCancel(b.ctx)
			defer cancel()
			b.epochHandler, err = b.initiateAndGetEpochHandler(ctx, epoch)
			if err != nil {
				return err
			}

			// get start slot for current epoch
			nextEpochStart, err := b.epochState.GetStartSlotForEpoch(epoch + 1)
			if err != nil {
				return fmt.Errorf("failed to get start slot for next epoch %d: %s", epoch+1, err)
			}

			nextEpochStartTime := getSlotStartTime(nextEpochStart, b.constants.slotDuration)
			epochTimer := time.NewTimer(time.Until(nextEpochStartTime))
			cleanup := func() {
				if !epochTimer.Stop() {
					<-epochTimer.C
				}
			}

			errCh := make(chan error)
			go b.epochHandler.run(errCh)

			select {
			case <-b.ctx.Done():
				cleanup()
				return b.ctx.Err()
			case <-b.pause:
				cleanup()
				return errServicePaused
			case <-epochTimer.C:
				// stop current epoch handler
				cancel()
			case err := <-errCh:
				logger.Errorf("error from epochHandler: %s", err)
			}

			// setup next epoch, re-invoke block authoring
			next, err := b.incrementEpoch()
			if err != nil {
				return fmt.Errorf("failed to increment epoch: %s", err)
			}

			logger.Infof("epoch %d complete, upcoming epoch: %d", epoch, next)
			epoch = next
			return nil
		}()
		if err != nil {
			if errors.Is(err, errServicePaused) || errors.Is(err, context.Canceled) {
				return nil
			}

			return err
		}
	}
}

func (b *Service) handleSlot(epoch, slotNum uint64, authorityIndex uint32, proof *VrfOutputAndProof) error {
	parentHeader, err := b.blockState.BestBlockHeader()
	if err != nil {
		logger.Errorf("block authoring: %s", err)
		return err
	}

	if parentHeader == nil {
		logger.Errorf("block authoring: %s", errNilParentHeader)
		return errNilParentHeader
	}

	// there is a chance that the best block header may change in the course of building the block,
	// so let's copy it first.
	parent, err := parentHeader.DeepCopy()
	if err != nil {
		return err
	}

	currentSlot := Slot{
		start:    time.Now(),
		duration: b.constants.slotDuration,
		number:   slotNum,
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

	hash := parent.Hash()
	rt, err := b.blockState.GetRuntime(&hash)
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)

	block, err := b.buildBlock(parent, currentSlot, rt, authorityIndex, proof)
	if err != nil {
		return err
	}

	logger.Infof(
		"built block %d with hash %s, state root %s, epoch %d and slot %d",
		block.Header.Number, block.Header.Hash(), block.Header.StateRoot, epoch, slotNum)
	logger.Tracef(
		"built block with parent hash %s, header %s and body %s",
		parent.Hash(), block.Header.String(), block.Body)

	b.telemetry.SendMessage(
		telemetry.NewPreparedBlockForProposing(
			block.Header.Hash(),
			block.Header.Number.String(),
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
