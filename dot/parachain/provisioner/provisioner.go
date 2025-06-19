// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package provisioner

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	prospectiveparachain "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/messages"
	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-provisioner"))

// inherentPreProposeTimeout is the duration to wait before inherent data is ready
const inherentPreProposeTimeout = 2 * time.Second

// maxDisputesVotes is the maximum number of disputes that Provisioner will include
// in the inherent data to prevent flooding the Runtime with excessive data.
//
// NOTE:
//   - Production value: 200,000.
//   - Test value: 200
const maxDisputesVotes = 200000

// voteSelectionBatchSize controls how many dispute votes to fetch from the dispute-coordinator per iteration
// in vote selection. Votes are fetched in batches until maxDisputesVotes is reached. This prevents
// an unnecessary load on the dispute-coordinator by avoiding fetching votes that won't be used.
//
// This value should be less than maxDisputesVotes. Increase it if the provisioner sends too many
// QueryCandidateVotes messages to the dispute-coordinator.
//
// NOTE:
//   - Production value: 1100.
//   - Test value: 11
const voteSelectionBatchSize = 1100

func New(overseerChan chan<- any, blockState BlockState) *Provisioner {
	return &Provisioner{
		subSystemToOverseer: overseerChan,
		blockState:          blockState,
		perRelayParent:      make(map[common.Hash]*perRelayParent),

		// Buffer size 2: one active delay + safety margin
		availableInherent: make(chan common.Hash, 2),
	}
}

type Provisioner struct {
	subSystemToOverseer chan<- any
	blockState          BlockState
	perRelayParent      map[common.Hash]*perRelayParent

	// TODO #4162
	// This doesn't have to be a channel with buffer.
	// The idea is to send a relay parent hash on this channel after INHERENT_TIMEOUT, open to design changes
	availableInherent chan common.Hash
}

func (p *Provisioner) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-overseerToSubSystem:
			if !ok {
				return
			}
			err := p.processMessage(msg)
			if err != nil {
				logger.Errorf("processing overseer message: %s", err)
			}
		case <-p.availableInherent:
			// This inherentAfterDelay gets populated while handling active leaves update signal
			// TODO #4162
		}
	}
}

func (p *Provisioner) processMessage(msg any) error {
	switch msg := msg.(type) {
	case parachaintypes.ActiveLeavesUpdateSignal:
		err := p.ProcessActiveLeavesUpdateSignal(msg)
		if err != nil {
			logger.Errorf("processing active leaves update signal: %s", err)
		}
	case provisionermessages.RequestInherentData:
		return p.requestInherentData(msg)
	case provisionermessages.ProvisionableData:
		p.processProvisionableData(msg)
	default:
		return parachaintypes.ErrUnknownOverseerMessage
	}

	return nil
}

func (*Provisioner) Name() parachaintypes.SubSystemName {
	return parachaintypes.Provisioner
}

func (p *Provisioner) ProcessActiveLeavesUpdateSignal(update parachaintypes.ActiveLeavesUpdateSignal) error {
	for _, deactivated := range update.Deactivated {
		delete(p.perRelayParent, deactivated)
	}

	if update.Activated != nil {
		p.perRelayParent[update.Activated.Hash] = &perRelayParent{leaf: update.Activated}

		go func() {
			time.Sleep(inherentPreProposeTimeout)
			p.availableInherent <- update.Activated.Hash
		}()
	}
	return nil
}

func (*Provisioner) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	// nothing to do here
	return nil
}

func (*Provisioner) Stop() {}

func (p *Provisioner) processProvisionableData(provisionableData provisionermessages.ProvisionableData) {
	state, exists := p.perRelayParent[provisionableData.RelayParent]
	if !exists {
		return
	}

	// note provisionable data
	switch data := provisionableData.Data.(type) {
	case provisionermessages.ProvisionableDataBitfield:
		state.signedBitfields = append(state.signedBitfields, data.Bitfield)
	case provisionermessages.ProvisionableDataMisbehaviorReport:
		// We choose not to punish these forms of misbehaviour for the time being.
		// Risks from misbehaviour are sufficiently mitigated at the protocol level
		// via reputation changes. Punitive actions here may become desirable
		// enough to dedicate time to in the future.
	}
}

func (p *Provisioner) requestInherentData(msg provisionermessages.RequestInherentData) error {
	perRP, exists := p.perRelayParent[msg.RelayParent]
	if !exists {
		return nil
	}

	if !perRP.isInherentReady {
		perRP.awaitingInherent = append(perRP.awaitingInherent, msg.ProvisionerInherentData)
		return nil
	}

	responseSenders := []chan provisionermessages.ProvisionerInherentData{msg.ProvisionerInherentData}
	return p.sendInherentData(perRP.leaf, perRP.signedBitfields, responseSenders)
}

func (p *Provisioner) sendInherentData(
	leaf *parachaintypes.ActivatedLeaf,
	signedBitfields []parachaintypes.CheckedSignedAvailabilityBitfield,
	responseSenders []chan provisionermessages.ProvisionerInherentData,
) error {
	instance, err := p.blockState.GetRuntime(leaf.Hash)
	if err != nil {
		return fmt.Errorf("getting runtime for leaf %s: %w", leaf.Hash, err)
	}

	cores, err := instance.ParachainHostAvailabilityCores()
	if err != nil {
		return fmt.Errorf("getting cores for leaf %s: %w", leaf.Hash, err)
	}

	disputes := SelectDisputes(p.subSystemToOverseer, p.blockState, leaf, maxDisputesVotes, voteSelectionBatchSize)

	bitfields, err := selectAvailabilityBitfields(cores, signedBitfields)
	if err != nil {
		return fmt.Errorf("selecting availability bitfields for leaf %s: %w", leaf.Hash, err)
	}

	candidates, err := p.selectCandidates(cores, bitfields, leaf)
	if err != nil {
		return fmt.Errorf("selecting candidates for leaf %s: %w", leaf.Hash, err)
	}

	inherentData := provisionermessages.ProvisionerInherentData{
		Bitfield:         bitfields,
		BackedCandidates: candidates,
		Disputes:         disputes,
	}

	for _, responseSender := range responseSenders {
		responseSender <- inherentData
	}
	return nil
}

func selectAvailabilityBitfields(
	cores []parachaintypes.CoreState,
	bitfields []parachaintypes.CheckedSignedAvailabilityBitfield,
) ([]parachaintypes.CheckedSignedAvailabilityBitfield, error) {
	coresLen := len(cores)

	// Map to keep the best bitfield per validator
	selected := make(map[parachaintypes.ValidatorIndex]parachaintypes.CheckedSignedAvailabilityBitfield)

	// Process each bitfield
	for _, bitfield := range bitfields {
		// Check if bitfield has the correct length matching cores
		if bitfield.Payload.Len() != coresLen {
			logger.Warnf("bitfield has incorrect length %d, expected %d",
				bitfield.Payload.Len(), coresLen)
			continue
		}

		// Check if this is the best bitfield from this validator
		current, exists := selected[bitfield.ValidatorIndex]
		if exists {
			// Compare bit counts - keep the one with more 1s
			if current.Payload.CountOnes() >= bitfield.Payload.CountOnes() {
				// dropping bitfield due to duplication - the better one is kept
				continue
			}
		}

		var continueOuterLoop bool

		// Check that bits aren't set for unoccupied cores
		for i, core := range cores {
			_, coreValue, err := core.IndexValue()
			if err != nil {
				return nil, fmt.Errorf("getting core index and value: %w", err)
			}
			_, isOccupied := coreValue.(parachaintypes.OccupiedCore)

			currBit, err := bitfield.Payload.Get(uint(i))
			if err != nil {
				return nil, fmt.Errorf("getting bit from bitfield: %w", err)
			}

			if !isOccupied && currBit {
				// Bit is set for an unoccupied core - invalid
				continueOuterLoop = true
				break
			}
		}

		if continueOuterLoop {
			continue
		}

		// This is the best valid bitfield from this validator so far
		selected[bitfield.ValidatorIndex] = bitfield
	}

	// Convert a map to slice, ordered by validator index
	result := make([]parachaintypes.CheckedSignedAvailabilityBitfield, 0, len(selected))

	keys := maps.Keys(selected)
	slices.Sort(keys)

	// Append values in order of sorted keys
	for _, k := range keys {
		result = append(result, selected[k])
	}

	return result, nil
}

func (p *Provisioner) selectCandidates(
	availabilityCores []parachaintypes.CoreState,
	bitfields []parachaintypes.CheckedSignedAvailabilityBitfield,
	leaf *parachaintypes.ActivatedLeaf,
) ([]parachaintypes.BackedCandidate, error) {
	selectedCandidates, err := p.requestBackableCandidates(availabilityCores, bitfields, leaf)
	if err != nil {
		return nil, fmt.Errorf("requesting backable candidates: %w", err)
	}

	getBackable := backing.GetBackableCandidatesMessage{
		Candidates: selectedCandidates,
		ResCh:      make(chan map[parachaintypes.ParaID][]*parachaintypes.BackedCandidate, len(selectedCandidates)),
	}

	var backableCandidates map[parachaintypes.ParaID][]*parachaintypes.BackedCandidate

	p.subSystemToOverseer <- getBackable
	select {
	case <-time.After(parachaintypes.SubsystemRequestTimeout):
		return nil, parachaintypes.ErrSubsystemRequestTimeout
	case backableCandidates = <-getBackable.ResCh:
	}

	// keep only one candidate with validation code.
	withValidationCode := false
	// merge the candidates into a common collection, preserving the order
	mergedCandidates := make([]parachaintypes.BackedCandidate, 0, len(availabilityCores))

	for _, paraCandidates := range backableCandidates {
		for _, candidate := range paraCandidates {
			if candidate.Candidate.Commitments.NewValidationCode != nil {
				if withValidationCode {
					// if we already have a candidate with validation code, break the loop
					break
				} else {
					withValidationCode = true
				}
			}

			mergedCandidates = append(mergedCandidates, *candidate)
		}
	}

	return mergedCandidates, nil
}

func (p *Provisioner) requestBackableCandidates(
	availabilityCores []parachaintypes.CoreState,
	bitfields []parachaintypes.CheckedSignedAvailabilityBitfield,
	relayParent *parachaintypes.ActivatedLeaf,
) (map[parachaintypes.ParaID][]*parachaintypes.CandidateHashAndRelayParent, error) {
	blockNumberUnderConstruction := parachaintypes.BlockNumber(relayParent.Number + 1)

	scheduledCoresPerPara := make(map[parachaintypes.ParaID]uint)
	ancestorsPerPara := make(map[parachaintypes.ParaID]prospectiveparachain.Ancestors, len(availabilityCores))

	for coreIdx, core := range availabilityCores {
		coreValue, err := core.Value()
		if err != nil {
			return nil, fmt.Errorf("getting core value: %w", err)
		}

		switch coreValue := coreValue.(type) {
		case parachaintypes.ScheduledCore:
			numOfCores, exists := scheduledCoresPerPara[coreValue.ParaID]
			if exists {
				scheduledCoresPerPara[coreValue.ParaID] = numOfCores + 1
			} else {
				scheduledCoresPerPara[coreValue.ParaID] = 1
			}

		case parachaintypes.OccupiedCore:
			isAvailable, err := bitfieldsIndicateAvailability(coreIdx, bitfields, coreValue.Availability)
			if err != nil {
				return nil, fmt.Errorf("checking availability for core %d: %w", coreIdx, err)
			}

			switch {
			case isAvailable:
				ancestors, exists := ancestorsPerPara[coreValue.CandidateDescriptor.ParaID]
				if !exists {
					ancestors = make(prospectiveparachain.Ancestors)
				}
				ancestors[parachaintypes.CandidateHash{Value: coreValue.CandidateHash}] = struct{}{}
				ancestorsPerPara[coreValue.CandidateDescriptor.ParaID] = ancestors

				if scheduledCore := coreValue.NextUpOnAvailable; scheduledCore != nil {
					numOfCores, exists := scheduledCoresPerPara[scheduledCore.ParaID]
					if exists {
						scheduledCoresPerPara[scheduledCore.ParaID] = numOfCores + 1
					} else {
						scheduledCoresPerPara[scheduledCore.ParaID] = 1
					}
				}
			case coreValue.TimeoutAt <= blockNumberUnderConstruction: // Timed out before being available.
				if scheduledCore := coreValue.NextUpOnTimeOut; scheduledCore != nil {
					numOfCores, exists := scheduledCoresPerPara[scheduledCore.ParaID]
					if exists {
						scheduledCoresPerPara[scheduledCore.ParaID] = numOfCores + 1
					} else {
						scheduledCoresPerPara[scheduledCore.ParaID] = 1
					}
				}
			default: // Not timed out and not available.
				ancestors, exists := ancestorsPerPara[coreValue.CandidateDescriptor.ParaID]
				if !exists {
					ancestors = make(prospectiveparachain.Ancestors)
				}
				ancestors[parachaintypes.CandidateHash{Value: coreValue.CandidateHash}] = struct{}{}
				ancestorsPerPara[coreValue.CandidateDescriptor.ParaID] = ancestors
			}

		case parachaintypes.Free:
			continue
		}
	}

	selectedCandidate := make(map[parachaintypes.ParaID][]*parachaintypes.CandidateHashAndRelayParent,
		len(scheduledCoresPerPara))

	orderedParaIds := maps.Keys(scheduledCoresPerPara)
	slices.Sort(orderedParaIds)

	for _, paraId := range orderedParaIds {
		coreCount := scheduledCoresPerPara[paraId]

		paraAncestor, exists := ancestorsPerPara[paraId]
		if exists {
			delete(ancestorsPerPara, paraId)
		}

		getBackable := prospectiveparachain.GetBackableCandidates{
			RelayParentHash: relayParent.Hash,
			ParaId:          paraId,
			RequestedQty:    uint32(coreCount),
			Ancestors:       paraAncestor,
			Response:        make(chan []*parachaintypes.CandidateHashAndRelayParent),
		}

		p.subSystemToOverseer <- getBackable
		select {
		case candidates := <-getBackable.Response:
			if len(candidates) == 0 {
				continue
			}
			selectedCandidate[paraId] = candidates
		case <-time.After(parachaintypes.SubsystemRequestTimeout):
			return nil, parachaintypes.ErrSubsystemRequestTimeout
		}
	}

	return selectedCandidate, nil
}

func bitfieldsIndicateAvailability(
	coreIndex int,
	bitfields []parachaintypes.CheckedSignedAvailabilityBitfield,
	availability parachaintypes.BitVec,
) (bool, error) {
	for _, bitfield := range bitfields {
		validatorIndex := uint(bitfield.ValidatorIndex)
		availabilityBit, err := availability.Get(validatorIndex)
		if err != nil {
			return false, nil //nolint:nilerr
		}

		bitfieldLength := bitfield.Payload.Len()
		if coreIndex < bitfieldLength {
			// impossible to get an error here, as we already ensure coreIndex is within bounds
			bit, _ := bitfield.Payload.Get(uint(coreIndex))
			availabilityBit = availabilityBit || bit

			err := availability.Set(validatorIndex, availabilityBit)
			if err != nil {
				return false, fmt.Errorf("setting availability bit: %w", err)
			}
		}
	}

	return 3*availability.CountOnes() >= 2*availability.Len(), nil
}

type perRelayParent struct {
	leaf             *parachaintypes.ActivatedLeaf
	signedBitfields  []parachaintypes.CheckedSignedAvailabilityBitfield
	isInherentReady  bool
	awaitingInherent []chan provisionermessages.ProvisionerInherentData
}
