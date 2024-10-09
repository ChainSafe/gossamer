package provisionermessages

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	_ Data = (*ProvisionableDataBackedCandidate)(nil)
	_ Data = (*ProvisionableDataMisbehaviorReport)(nil)
)

type RequestInherentData struct {
	RelayParent             common.Hash
	ProvisionerInherentData chan ProvisionerInherentData
}

type ProvisionerInherentData struct {
}

// ProvisionableData is a provisioner message.
// This data should become part of a relay chain block.
type ProvisionableData struct {
	RelayParent common.Hash
	Data        Data
}

// Data becomes intrinsics or extrinsics which should be included in a future relay chain block.
type Data interface {
	IsData()
}

// ProvisionableDataBackedCandidate is a provisionable data.
// The Candidate Backing subsystem believes that this candidate is valid, pending availability.
type ProvisionableDataBackedCandidate parachaintypes.CandidateReceipt

func (ProvisionableDataBackedCandidate) IsData() {}

// ProvisionableDataMisbehaviorReport represents self-contained proofs of validator misbehaviour.
type ProvisionableDataMisbehaviorReport struct {
	ValidatorIndex parachaintypes.ValidatorIndex
	Misbehaviour   parachaintypes.Misbehaviour
}

func (ProvisionableDataMisbehaviorReport) IsData() {}
