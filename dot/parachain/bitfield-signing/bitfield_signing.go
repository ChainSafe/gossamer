// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package bitfieldsigning

import (
	"context"
	"fmt"
	"sync"
	"time"

	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

const availabilityDistributionWaitingPeriod = time.Millisecond * 1500

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-bitfield-signing"))

type signingTask struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// BitfieldSigning is the parachain subsystem that validators vote on the availability of a
// backed candidate by issuing signed bitfields
type BitfieldSigning struct {
	subSystemToOverseer chan<- any
	keystore            keystore.Keystore
	blockState          BlockState
	signingTasksTracker map[common.Hash]*signingTask
}

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

// NewBitfieldSigning creates a new BitfieldSigning subsystem
func NewBitfieldSigning(overseerChan chan<- any, ks keystore.Keystore, blockState BlockState) *BitfieldSigning {
	return &BitfieldSigning{
		subSystemToOverseer: overseerChan,
		keystore:            ks,
		blockState:          blockState,
		signingTasksTracker: make(map[common.Hash]*signingTask),
	}
}

// Run starts the BitfieldSigning subsystem
func (b *BitfieldSigning) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubSystem:
			err := b.processMessage(msg)
			if err != nil {
				logger.Errorf("processing message: %s", err.Error())
			}
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

// processMessage processes messages sent to the BitfieldSigning subsystem
func (b *BitfieldSigning) processMessage(msg any) error {
	switch msg := msg.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		err := b.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			return fmt.Errorf("processing active leaves update signal: %w", err)
		}
	case parachaintypes.BlockFinalizedSignal:
		return b.ProcessBlockFinalizedSignal(msg)
	default:
		return fmt.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}
	return nil
}

// Name returns the name of the subsystem
func (b *BitfieldSigning) Name() parachaintypes.SubSystemName {
	return parachaintypes.BitfieldSigning
}

// ProcessActiveLeavesUpdateSignal processes active leaves update signal
func (b *BitfieldSigning) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	// cancel and remove jobs for deactivated leaves
	for _, hash := range signal.Deactivated {
		task := b.signingTasksTracker[hash]
		if task != nil {
			task.cancel()
		}
		delete(b.signingTasksTracker, hash)
	}

	activatedLeaf := signal.Activated
	if activatedLeaf == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := handleActiveLeavesUpdate(ctx, b, activatedLeaf)
		if err != nil {
			logger.Errorf("handleActiveLeavesUpdate error: %s", err)
		}
	}()
	b.signingTasksTracker[activatedLeaf.Hash] = &signingTask{ctx, cancel}

	return nil
}

func handleActiveLeavesUpdate(ctx context.Context, b *BitfieldSigning, activatedLeaf *parachaintypes.ActivatedLeaf,
) error {
	relayParent := activatedLeaf.Hash
	rt, err := b.blockState.GetRuntime(relayParent)
	if err != nil {
		return fmt.Errorf("getting runtime: %w", err)
	}

	// get validators info
	validators, err := rt.ParachainHostValidators()
	if err != nil {
		return fmt.Errorf("getting validators: %w", err)

	}
	validatorID, validatorIndex := parachainutil.SigningKeyAndIndex(validators, b.keystore)
	if validatorID == nil {
		// skip the logic if not a validator node
		return nil
	}

	sessionIndex, err := rt.ParachainHostSessionIndexForChild()
	if err != nil {
		return fmt.Errorf("fetching parachain host session index data: %w", err)
	}
	signingContext := parachaintypes.SigningContext{
		SessionIndex: sessionIndex,
		ParentHash:   relayParent,
	}
	validator := parachaintypes.Validator{
		Key:            *validatorID,
		Index:          validatorIndex,
		SigningContext: signingContext,
	}

	// wait for availability distribution has the chance to make candidates available.
	time.Sleep(availabilityDistributionWaitingPeriod)

	// construct the bitfield according to the availability store
	bitfield, err := constructAvailabilityBitfield(ctx, rt, validatorIndex, b.subSystemToOverseer)
	if err != nil {
		return fmt.Errorf("construct availabilityBitfield: %w", err)
	}

	// signing process
	data, err := bitfield.MarshalSCALE()
	if err != nil {
		return fmt.Errorf("marshal bitfield for signing: %w", err)
	}
	signature, err := validator.Sign(b.keystore, data)
	if err != nil {
		return fmt.Errorf("signing bitfield: %w", err)
	}

	// distribute to subsystem to overseer chan
	b.subSystemToOverseer <- parachaintypes.DistributeBitfield{
		RelayParent: relayParent,
		Bitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        bitfield,
			ValidatorIndex: validatorIndex,
			Signature:      *signature,
		},
	}

	return nil
}

// ProcessBlockFinalizedSignal processes block finalized signal
func (b *BitfieldSigning) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	return nil
}

// Stop stops the BitfieldSigning subsystem
func (b *BitfieldSigning) Stop() {
	logger.Infof("Stopping BitfieldSigning subsystem")
}

func constructAvailabilityBitfield(
	ctx context.Context,
	rt runtime.Instance,
	validatorIdx parachaintypes.ValidatorIndex,
	subSystemToOverseer chan<- any,
) (parachaintypes.BitVec, error) {
	cores, err := rt.ParachainHostAvailabilityCores()
	if err != nil {
		return parachaintypes.BitVec{}, fmt.Errorf("querying availability cores: %w", err)
	}

	// init a bitfield without caring the order and also allocate all indices with false to avoid race condition later
	bitfield := make([]bool, len(cores))

	var wg sync.WaitGroup
	for coreOrderIndex, core := range cores {
		v, err := core.Value()
		if err != nil {
			return parachaintypes.BitVec{}, err
		}
		value, ok := v.(parachaintypes.OccupiedCore)
		if !ok {
			bitfield[coreOrderIndex] = false
			continue
		}
		wg.Add(1)
		go func(oi int, v parachaintypes.OccupiedCore) {
			defer wg.Done()

			receivingChan := make(chan bool)
			queryPayload := availabilitystore.QueryChunkAvailability{
				CandidateHash:  parachaintypes.CandidateHash{Value: v.CandidateHash},
				ValidatorIndex: uint32(validatorIdx),
				Sender:         receivingChan,
			}

			// send QueryChunkAvailability to availability store via overseer
			subSystemToOverseer <- queryPayload

			// check if cancel signal is coming
			// we can't cancel the process anytime during the spawned goroutines
			// instead we block the process here so that either cancel or receivingChan responses
			var data bool
			select {
			case <-ctx.Done():
				logger.Infof("process for handleActiveLeavesUpdate is abort due to deactivated leaves")
				return
			case res := <-receivingChan:
				data = res
			}
			bitfield[oi] = data
		}(coreOrderIndex, value)
	}
	wg.Wait()

	return parachaintypes.NewBitVec(bitfield)
}
