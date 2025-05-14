// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package provisionermessages

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	_ Data = (*ProvisionableDataBitfield)(nil)
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

// ProvisionableDataMisbehaviorReport represents self-contained proofs of validator misbehaviour.
type ProvisionableDataMisbehaviorReport struct {
	ValidatorIndex parachaintypes.ValidatorIndex
	Misbehaviour   parachaintypes.Misbehaviour
}

func (ProvisionableDataMisbehaviorReport) IsData() {}

// ProvisionableDataBitfield indicates the availability of various candidate blocks.
type ProvisionableDataBitfield struct {
	RelayParent common.Hash
	Bitfield    parachaintypes.CheckedSignedAvailabilityBitfield
}

func (ProvisionableDataBitfield) IsData() {}
