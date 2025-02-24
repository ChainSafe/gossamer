// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package bitfield_signing

import (
	"context"
	"fmt"
	availabilitystore "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"sort"
	"sync"
	"time"
)

const availabilityDistributionWaitingPeriod = time.Millisecond * 1500

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-bitfield-signing"))

type signingTask struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type bitfieldData struct {
	index int
	data  bool
}

// BitfieldSigning is the parachain subsystem that validators vote on the availability of a
// backed candidate by issuing signed bitfields
type BitfieldSigning struct {
	subSystemToOverseer chan<- any
	keystore            keystore.Keystore
	blockState          BlockState
	signingTasksTracker map[common.Hash]signingTask
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
		signingTasksTracker: make(map[common.Hash]signingTask),
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
		b.signingTasksTracker[hash].cancel()
		delete(b.signingTasksTracker, hash)
	}

	activatedLeaf := signal.Activated
	if activatedLeaf == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			logger.Infof("process for handleActiveLeavesUpdate is done due to deactivated leaves")
			return
		default:
			b.handleActiveLeavesUpdate(ctx, activatedLeaf)
		}
	}(ctx)

	b.signingTasksTracker[activatedLeaf.Hash] = signingTask{ctx, cancel}

	return nil
}

func (b *BitfieldSigning) handleActiveLeavesUpdate(ctx context.Context, activatedLeaf *parachaintypes.ActivatedLeaf) {
	relayParent := activatedLeaf.Hash
	rt, err := b.blockState.GetRuntime(relayParent)
	if err != nil {
		logger.Errorf("getting runtime: %s", err)
		return
	}

	// get validators info
	validators, err := rt.ParachainHostValidators()
	if err != nil {
		logger.Errorf("getting validators: %s", err)
		return
	}
	validatorID, validatorIndex := parachainutil.SigningKeyAndIndex(validators, b.keystore)
	if validatorID == nil {
		// skip the logic if not a validator node
		return
	}

	// wait for availability distribution has the chance to make candidates available.
	time.Sleep(availabilityDistributionWaitingPeriod)

	// construct the bitfield according to the availability store
	bitfield, err := constructAvailabilityBitfield(ctx, rt, validatorIndex, b.subSystemToOverseer)
	if err != nil {
		logger.Errorf("construct availabilityBitfield: %s", err)
		return
	}

	// signing process
	data, err := bitfield.MarshalSCALE()
	if err != nil {
		logger.Errorf("marshal bitfield for signing: %s", err)
		return
	}
	validatorPublicKey, err := sr25519.NewPublicKey(validatorID[:])
	if err != nil {
		logger.Errorf("getting signer's public key: %s", err)
		return
	}
	signatureBytes, err := b.keystore.GetKeypair(validatorPublicKey).Sign(data)
	if err != nil {
		logger.Errorf("signing bitfield: %s", err)
		return
	}
	var signature parachaintypes.Signature
	copy(signature[:], signatureBytes)

	// distribute to subsystem to overseer chan
	b.subSystemToOverseer <- parachaintypes.DistributeBitfield{
		RelayParent: relayParent,
		Bitfield: parachaintypes.UncheckedSignedAvailabilityBitfield{
			Payload:        bitfield,
			ValidatorIndex: validatorIndex,
			Signature:      parachaintypes.ValidatorSignature(signature),
		},
	}

	return
}

// ProcessBlockFinalizedSignal processes block finalized signal
func (b *BitfieldSigning) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	return nil
}

// Stop stops the BitfieldSigning subsystem
func (b *BitfieldSigning) Stop() {
	// TODO: anything to do to stop this subsystem ?
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

	// init a bitfield without caring the order
	bitfield := make([]bitfieldData, 0, len(cores))

	var wg sync.WaitGroup
	for coreOrderIndex, core := range cores {
		coreValueIndex, value, err := core.IndexValue()
		if err != nil {
			return parachaintypes.BitVec{}, err
		}
		// 0 is type of parachaintypes.OccupiedCore
		if coreValueIndex == 0 {
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

				// append the result to the bitfield
				bitfield = append(bitfield, bitfieldData{index: oi, data: <-receivingChan})
			}(coreOrderIndex, value.(parachaintypes.OccupiedCore))
		} else {
			bitfield = append(bitfield, bitfieldData{index: coreOrderIndex, data: false})
		}
	}
	wg.Wait()

	return bitfieldOrderGuard(bitfield), nil
}

// bitfieldOrderGuard sort the fulfilled bitfield data in its correct order according to the CoreState and return
// the bitfield
func bitfieldOrderGuard(bitfieldData []bitfieldData) parachaintypes.BitVec {
	sort.Slice(bitfieldData, func(i, j int) bool {
		return bitfieldData[i].index < bitfieldData[j].index
	})

	b := make([]bool, 0, len(bitfieldData))
	for _, item := range bitfieldData {
		b = append(b, item.data)
	}

	return parachaintypes.NewBitVec(b)
}
