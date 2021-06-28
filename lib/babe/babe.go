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

package babe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/runtime"
	log "github.com/ChainSafe/log15"
	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

var (
	logger          log.Logger
	initialWaitTime time.Duration
)

// Service contains the VRF keys for the validator, as well as BABE configuation data
type Service struct {
	ctx       context.Context
	cancel    context.CancelFunc
	authority bool
	dev       bool

	// Storage interfaces
	blockState       BlockState
	storageState     StorageState
	transactionState TransactionState
	epochState       EpochState
	epochLength      uint64

	blockImportHandler BlockImportHandler

	// BABE authority keypair
	keypair *sr25519.Keypair // TODO: change to BABE keystore

	// Epoch configuration data
	slotDuration time.Duration
	epochData    *epochData
	slotToProof  map[uint64]*VrfOutputAndProof // for slots where we are a producer, store the vrf output (bytes 0-32) + proof (bytes 32-96)

	// State variables
	sync.RWMutex
	pause chan struct{}
}

// ServiceConfig represents a BABE configuration
type ServiceConfig struct {
	LogLvl               log.Lvl
	BlockState           BlockState
	StorageState         StorageState
	TransactionState     TransactionState
	EpochState           EpochState
	BlockImportHandler   BlockImportHandler
	Keypair              *sr25519.Keypair
	Runtime              runtime.Instance
	AuthData             []*types.Authority
	IsDev                bool
	ThresholdNumerator   uint64 // for development purposes
	ThresholdDenominator uint64 // for development purposes
	SlotDuration         uint64 // for development purposes; in milliseconds
	EpochLength          uint64 // for development purposes; in slots
	Authority            bool
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

	logger = log.New("pkg", "babe")
	h := log.StreamHandler(os.Stdout, log.TerminalFormat())
	h = log.CallerFileHandler(h)
	logger.SetHandler(log.LvlFilterHandler(cfg.LogLvl, h))

	ctx, cancel := context.WithCancel(context.Background())

	babeService := &Service{
		ctx:                ctx,
		cancel:             cancel,
		blockState:         cfg.BlockState,
		storageState:       cfg.StorageState,
		epochState:         cfg.EpochState,
		epochLength:        cfg.EpochLength,
		keypair:            cfg.Keypair,
		transactionState:   cfg.TransactionState,
		slotToProof:        make(map[uint64]*VrfOutputAndProof),
		pause:              make(chan struct{}),
		authority:          cfg.Authority,
		dev:                cfg.IsDev,
		blockImportHandler: cfg.BlockImportHandler,
	}

	epoch, err := cfg.EpochState.GetCurrentEpoch()
	if err != nil {
		return nil, err
	}

	err = babeService.setupParameters(cfg)
	if err != nil {
		return nil, err
	}

	initialWaitTime = babeService.slotDuration * 5

	logger.Debug("created service",
		"epoch", epoch,
		"block producer", cfg.Authority,
		"slot duration", babeService.slotDuration,
		"epoch length (slots)", babeService.epochLength,
		"authorities", Authorities(babeService.epochData.authorities),
		"authority index", babeService.epochData.authorityIndex,
		"threshold", babeService.epochData.threshold,
		"randomness", babeService.epochData.randomness,
	)
	return babeService, nil
}

func (b *Service) setupParameters(cfg *ServiceConfig) error {
	var err error
	b.epochData = &epochData{}

	epochData, err := b.epochState.GetLatestEpochData()
	if err != nil {
		return err
	}

	b.epochData.randomness = epochData.Randomness

	configData, err := b.epochState.GetLatestConfigData()
	if err != nil {
		return err
	}

	// if slot duration is set via the config file, overwrite the runtime value
	switch {
	case cfg.SlotDuration > 0 && cfg.IsDev: // TODO: remove this, needs to be set via runtime
		b.slotDuration, err = time.ParseDuration(fmt.Sprintf("%dms", cfg.SlotDuration))
	case cfg.SlotDuration > 0 && !cfg.IsDev:
		err = errors.New("slot duration modified in config for non-dev chain")
	default:
		b.slotDuration, err = b.epochState.GetSlotDuration()
	}
	if err != nil {
		return err
	}

	switch {
	case cfg.EpochLength != 0 && cfg.IsDev: // TODO: remove this, needs to be set via runtime
		b.epochLength = cfg.EpochLength
	case cfg.EpochLength > 0 && !cfg.IsDev:
		err = errors.New("epoch length modified in config for non-dev chain")
	default:
		b.epochLength, err = b.epochState.GetEpochLength()
	}
	if err != nil {
		return err
	}

	switch {
	case cfg.AuthData != nil && cfg.IsDev: // TODO: remove this, needs to be set via runtime
		b.epochData.authorities = cfg.AuthData
	case cfg.AuthData != nil && !cfg.IsDev:
		return errors.New("authority data modified in config for non-dev chain")
	default:
		b.epochData.authorities = epochData.Authorities
	}

	switch {
	case cfg.ThresholdDenominator != 0 && cfg.IsDev: // TODO: remove this, needs to be set via runtime
		b.epochData.threshold, err = CalculateThreshold(cfg.ThresholdNumerator, cfg.ThresholdDenominator, len(b.epochData.authorities))
	case cfg.ThresholdDenominator != 0 && !cfg.IsDev:
		err = errors.New("threshold modified in config for non-dev chain")
	default:
		b.epochData.threshold, err = CalculateThreshold(configData.C1, configData.C2, len(b.epochData.authorities))
	}
	if err != nil {
		return err
	}

	if !cfg.Authority {
		return nil
	}

	b.epochData.authorityIndex, err = b.getAuthorityIndex(b.epochData.authorities)
	return err
}

// Start starts BABE block authoring
func (b *Service) Start() error {
	if !b.authority {
		return nil
	}

	// wait a bit to check if we need to sync before initiating
	<-time.NewTimer(initialWaitTime).C

	go b.initiate()
	return nil
}

// SlotDuration returns the current service slot duration in milliseconds
func (b *Service) SlotDuration() uint64 {
	return uint64(b.slotDuration.Milliseconds())
}

// EpochLength returns the current service epoch duration
func (b *Service) EpochLength() uint64 {
	return b.epochLength
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
func (b *Service) Authorities() []*types.Authority {
	return b.epochData.authorities
}

// IsStopped returns true if the service is stopped (ie not producing blocks)
func (b *Service) IsStopped() bool {
	return b.ctx.Err() != nil
}

func (b *Service) getAuthorityIndex(Authorities []*types.Authority) (uint32, error) {
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
	return b.slotDuration
}

func (b *Service) initiate() {
	if b.blockState == nil {
		logger.Error("block authoring", "error", "blockState is nil")
		return
	}

	if b.storageState == nil {
		logger.Error("block authoring", "error", "storageState is nil")
		return
	}

	err := b.invokeBlockAuthoring()
	if err != nil {
		logger.Crit("block authoring error", "error", err)
	}
}

func (b *Service) invokeBlockAuthoring() error {
	epoch, err := b.epochState.GetCurrentEpoch()
	if err != nil {
		logger.Error("failed to get current epoch", "error", err)
		return err
	}

	for {
		err := b.initiateEpoch(epoch)
		if err != nil {
			logger.Error("failed to initiate epoch", "epoch", epoch, "error", err)
			return err
		}

		epochStartSlot, err := b.waitForEpochStart(epoch)
		if err != nil {
			logger.Error("failed to wait for epoch start", "epoch", epoch, "error", err)
			return err
		}

		// calculate current slot
		startSlot := getCurrentSlot(b.slotDuration)
		intoEpoch := startSlot - epochStartSlot

		// if the calculated amount of slots "into the epoch" is greater than the epoch length,
		// we've been offline for more than an epoch, and need to sync. pause BABE for now, syncer will
		// resume it when ready
		if b.epochLength <= intoEpoch && !b.dev {
			logger.Debug("pausing BABE, need to sync",
				"slots into epoch",
				intoEpoch, "startSlot",
				startSlot, "epochStart", epochStartSlot,
			)
			return b.Pause()
		}

		if b.dev {
			intoEpoch = intoEpoch % b.epochLength
		}

		logger.Info("current epoch", "epoch", epoch, "slots into epoch", intoEpoch)

		slotDone := make([]<-chan time.Time, b.epochLength-intoEpoch)
		for i := 0; i < int(b.epochLength-intoEpoch); i++ {
			slotDone[i] = time.After(b.getSlotDuration() * time.Duration(i))
		}

		for i := 0; i < int(b.epochLength-intoEpoch); i++ {
			select {
			case <-b.ctx.Done():
				return nil
			case <-b.pause:
				return nil
			case <-slotDone[i]:
				slotNum := startSlot + uint64(i)
				err = b.handleSlot(slotNum)
				if err == ErrNotAuthorized {
					logger.Debug("not authorized to produce a block in this slot",
						"epoch", epoch,
						"slot", slotNum,
						"slots into epoch", slotNum-epochStartSlot,
					)
					continue
				} else if err != nil {
					logger.Warn("failed to handle slot", "slot", slotNum, "error", err)
					continue
				}
			}
		}

		// setup next epoch, re-invoke block authoring
		next, err := b.incrementEpoch()
		if err != nil {
			logger.Error("failed to increment epoch", "error", err)
			return err
		}

		logger.Info("epoch complete!", "completed epoch", epoch, "upcoming epoch", next)
		epoch = next
	}
}

func (b *Service) waitForEpochStart(epoch uint64) (uint64, error) {
	// get start slot for current epoch
	epochStart, err := b.epochState.GetStartSlotForEpoch(epoch)
	if err != nil {
		logger.Error("failed to get start slot for current epoch", "epoch", epoch, "error", err)
		return 0, err
	}

	epochStartTime := getSlotStartTime(epochStart, b.slotDuration)
	logger.Debug("checking if epoch started", "epoch start", epochStartTime, "now", time.Now())

	// check if it's time to start the epoch yet. if not, wait until it is
	if time.Since(epochStartTime) < 0 {
		logger.Debug("waiting for epoch to start")
		err = func() error {
			timer := time.NewTimer(time.Until(epochStartTime))
			defer timer.Stop()
			select {
			case <-timer.C:
				return nil
			case <-b.ctx.Done():
				return errors.New("context cancelled")
			case <-b.pause:
				return errors.New("service paused")
			}
		}()

		if err != nil {
			return 0, err
		}
	}

	return epochStart, nil
}

func (b *Service) handleSlot(slotNum uint64) error {
	if _, has := b.slotToProof[slotNum]; !has {
		return ErrNotAuthorized
	}

	parentHeader, err := b.blockState.BestBlockHeader()
	if err != nil {
		logger.Error("block authoring", "error", err)
		return err
	}

	if parentHeader == nil {
		logger.Error("block authoring", "error", "parent header is nil")
		return err
	}

	// there is a chance that the best block header may change in the course of building the block,
	// so let's copy it first.
	parent := parentHeader.DeepCopy()

	currentSlot := Slot{
		start:    time.Now(),
		duration: b.slotDuration,
		number:   slotNum,
	}

	// set runtime trie before building block
	// if block building is successful, store the resulting trie in the storage state
	ts, err := b.storageState.TrieState(&parent.StateRoot)
	if err != nil || ts == nil {
		logger.Error("failed to get parent trie", "parent state root", parent.StateRoot, "error", err)
		return err
	}

	hash := parent.Hash()
	rt, err := b.blockState.GetRuntime(&hash)
	if err != nil {
		return err
	}

	rt.SetContextStorage(ts)

	block, err := b.buildBlock(parent, currentSlot, rt)
	if err != nil {
		return err
	}

	logger.Info("built block", "hash", block.Header.Hash().String(),
		"number", block.Header.Number,
		"state root", block.Header.StateRoot,
		"slot", slotNum,
	)
	logger.Debug("built block",
		"header", block.Header,
		"body", block.Body,
		"parent", parent.Hash(),
	)

	if err := b.blockImportHandler.HandleBlockProduced(block, ts); err != nil {
		logger.Warn("failed to import built block", "error", err)
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
