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
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"time"
)

const availabilityDistributionWaitingPeriod = time.Millisecond * 1500

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-bitfield-signing"))

// TODO: do we really need this subsystem state?
// signingTask is the subsystem state
//type signingTask struct {
//	ctx      context.Context
//	response <-chan parachaintypes.UncheckedSignedAvailabilityBitfield
//}

// BitfieldSigning is the parachain subsystem that validators vote on the availability of a
// backed candidate by issuing signed bitfields
type BitfieldSigning struct {
	subSystemToOverseer chan<- any
	keystore            keystore.Keystore
	//tasks               map[common.Hash]*signingTask
	bs BlockState
}

type BlockState interface {
	GetRuntime(blockHash common.Hash) (instance runtime.Instance, err error)
}

// NewBitfieldSigning creates a new BitfieldSigning subsystem
func NewBitfieldSigning(overseerChan chan<- any, ks keystore.Keystore, blockState BlockState) *BitfieldSigning {
	return &BitfieldSigning{
		subSystemToOverseer: overseerChan,
		keystore:            ks,
		//tasks:               make(map[common.Hash]*signingTask),
		bs: blockState,
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
			logger.Errorf("failed to process active leaves update signal: %w", err)
		}
	case parachaintypes.BlockFinalizedSignal:
		err := b.ProcessBlockFinalizedSignal(msg)
		if err != nil {
			logger.Errorf("failed to process block finalized signal: %w", err)
		}
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
	// cancel the task if leaves deactivated
	//for _, deactivatedLeaf := range signal.Deactivated {
	//	task := b.tasks[deactivatedLeaf]
	//	task.ctx.Done()
	//	delete(b.tasks, deactivatedLeaf)
	//}

	activatedLeaf := signal.Activated
	if activatedLeaf == nil {
		return nil
	}

	relayParent := activatedLeaf.Hash
	rt, err := b.bs.GetRuntime(relayParent)
	if err != nil {
		return err
	}

	// skip the logic if not a validator node
	// TODO: double check if runtime api is correct
	if !rt.Validator() {
		return nil
	}

	// wait for availability distribution has the chance to make candidates available.
	time.Sleep(availabilityDistributionWaitingPeriod)

	// get validator info
	// TODO: is this the right way to get the current validator index?
	validators, err := rt.ParachainHostValidators()
	if err != nil {
		return err
	}
	validatorID, validatorIndex := parachainutil.SigningKeyAndIndex(validators, b.keystore)

	// construct the bitfield according to the availability store
	bitfield, err := constructAvailabilityBitfield(rt, validatorIndex)
	if err != nil {
		return err
	}

	// sign the bitfield
	// TODO: Optimize the signing logic
	sessionIndex, err := rt.ParachainHostSessionIndexForChild()
	if err != nil {
		return err
	}
	signingContext := parachaintypes.SigningContext{
		SessionIndex: sessionIndex,
		ParentHash:   relayParent,
	}
	statement := parachaintypes.NewStatementVDT()
	err = statement.SetValue(bitfield)
	if err != nil {
		return err
	}
	signature, err := statement.Sign(b.keystore, signingContext, parachaintypes.ValidatorID(validatorID[:]))
	if err != nil {
		return err
	}

	// update the subsystem state
	signedBitfield := parachaintypes.UncheckedSignedAvailabilityBitfield{
		Payload:        parachaintypes.BitVec(*bitfield),
		ValidatorIndex: validatorIndex,
		Signature:      *signature,
	}
	//b.tasks[activatedLeaf.Hash] = &signingTask{
	//	ctx:      context.Background(),
	//	response: b.subSystemToOverseer,
	//}

	// distribute to subsystem to overseer chan
	b.subSystemToOverseer <- parachaintypes.DistributeBitfield{
		RelayParent: activatedLeaf.Hash,
		Bitfield:    signedBitfield,
	}

	return nil
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

type availabilityBitfield parachaintypes.BitVec

func constructAvailabilityBitfield(
	rt runtime.Instance,
	validatorIdx parachaintypes.ValidatorIndex,
) (*availabilityBitfield, error) {
	cores, err := rt.ParachainHostAvailabilityCores()
	if err != nil {
		return nil, err
	}

	// init a bitfield
	bitfield := make([]bool, len(cores))

	for _, core := range cores {
		index, value, err := core.IndexValue()
		if err != nil {
			return nil, err
		}
		// 0 is type of parachaintypes.OccupiedCore
		if index == 0 {
			c := value.(parachaintypes.OccupiedCore)

			receivingChan := make(chan bool)

			queryPayload := availabilitystore.QueryChunkAvailability{
				CandidateHash:  parachaintypes.CandidateHash{Value: c.CandidateHash},
				ValidatorIndex: uint32(validatorIdx),
				Sender:         receivingChan,
			}
			// TODO: send query to availability store via overseer

			// append the result to the bitfield
			bitfield = append(bitfield, <-receivingChan)
		} else {
			bitfield = append(bitfield, false)
		}

	}

	availBitfield := availabilityBitfield(parachaintypes.NewBitVec(bitfield))
	return &availBitfield, nil
}
