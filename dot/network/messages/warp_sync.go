// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package messages

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/client/consensus/grandpa"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// WarpProofRequest is a struct for p2p warp proof request
type WarpProofRequest struct {
	Begin common.Hash
}

func NewWarpProofRequest(from common.Hash) *WarpProofRequest {
	return &WarpProofRequest{
		Begin: from,
	}
}

// Decode decodes the message into a WarpProofRequest
func (wpr *WarpProofRequest) Decode(in []byte) error {
	return scale.Unmarshal(in, wpr)
}

// Encode encodes the warp sync request
func (wpr *WarpProofRequest) Encode() ([]byte, error) {
	if wpr == nil {
		return nil, fmt.Errorf("cannot encode nil WarpProofRequest")
	}
	return scale.Marshal(*wpr)
}

// String returns the string representation of a WarpProofRequest
func (wpr *WarpProofRequest) String() string {
	if wpr == nil {
		return "WarpProofRequest=nil"
	}

	return fmt.Sprintf("WarpProofRequest begin=%v", wpr.Begin)
}

type WarpSyncFragment struct {
	Header        types.Header
	Justification grandpa.GrandpaJustification[hash.H256, uint64]
}

type WarpSyncProof struct {
	Proofs []WarpSyncFragment
	// indicates whether the warp sync has been completed
	IsFinished   bool
	proofsLength int
}

func (wsp *WarpSyncProof) Decode(in []byte) error {
	return scale.Unmarshal(in, wsp)
}

func (wsp *WarpSyncProof) Encode() ([]byte, error) {
	if wsp == nil {
		return nil, fmt.Errorf("cannot encode nil WarpSyncProof")
	}
	return scale.Marshal(*wsp)
}

func (wsp *WarpSyncProof) String() string {
	if wsp == nil {
		return "WarpSyncProof=nil"
	}

	return fmt.Sprintf("WarpSyncProof proofs=%v isFinished=%v proofsLength=%v",
		wsp.Proofs, wsp.IsFinished, wsp.proofsLength)
}

var _ P2PMessage = (*WarpSyncProof)(nil)
var _ P2PMessage = (*WarpProofRequest)(nil)
