// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package messages

import (
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

const (
	// AttestedCandidateTimeout We want attested candidate requests to time out relatively fast,
	// because slow requests will bottleneck the backing system. Ideally, we'd have
	// an adaptive timeout based on the candidate size, because there will be a lot of variance
	// in candidate sizes: candidates with no code and no messages vs candidates with code
	// and messages.
	//
	// We supply leniency because there are often large candidates and asynchronous
	// backing allows them to be included over a longer window of time. Exponential back-off
	// up to a maximum of 10 seconds would be ideal, but isn't supported by the
	// infrastructure here yet: see https://github.com/paritytech/polkadot/issues/6009
	AttestedCandidateTimeout = time.Millisecond * 2500

	// MaxParallelAttestedCandidateRequests We don't want a slow peer to slow down all the others,
	// at the same time we want to get out the data quickly in full to at least some peers
	// (as this will reduce load on us as they then can start serving the data). So this value is a tradeoff.
	// 5 seems to be sensible. So we would need to have 5 slow nodes connected, to delay transfer for others
	// by [AttestedCandidateTimeout].
	MaxParallelAttestedCandidateRequests = 5
)

// AttestedCandidateRequest is a request for a candidate with statements.
type AttestedCandidateRequest struct {
	// Hash of the candidate we want to request.
	CandidateHash parachaintypes.CandidateHash
	// Statement filter with 'OR' semantics, indicating which validators
	// not to send statements for.
	//
	// The filter must have exactly the minimum size required to
	// fit all validators from the backing group.
	//
	// The response may not contain any statements masked out by this mask.
	Mask parachaintypes.StatementFilter
}

// Encode returns the SCALE encoding of the AttestedCandidateRequest
func (r *AttestedCandidateRequest) Encode() ([]byte, error) {
	return scale.Marshal(*r)
}

// Decode returns the SCALE decoding of the AttestedCandidateRequest
func (r *AttestedCandidateRequest) Decode(in []byte) (err error) {
	return scale.Unmarshal(in, r)
}

func (r *AttestedCandidateRequest) Response() network.ResponseMessage {
	return &AttestedCandidateResponse{}
}

func (r *AttestedCandidateRequest) Protocol() ReqProtocolName {
	return AttestedCandidateV2
}

// AttestedCandidateResponse is the response to an [AttestedCandidateRequest].
type AttestedCandidateResponse struct {
	CandidateReceipt        parachaintypes.CommittedCandidateReceiptV2
	PersistedValidationData parachaintypes.PersistedValidationData
	Statements              []any // TODO: implement parachaintypes.UncheckedSignedStatement
}

// Encode returns the SCALE encoding of the AttestedCandidateResponse
func (r *AttestedCandidateResponse) Encode() ([]byte, error) {
	return scale.Marshal(*r)
}

// Decode returns the SCALE decoding of the AttestedCandidateResponse
func (r *AttestedCandidateResponse) Decode(in []byte) (err error) {
	return scale.Unmarshal(in, r)
}

// String returns a human-readable representation of the AttestedCandidateResponse
func (r *AttestedCandidateResponse) String() string {
	return fmt.Sprintf(
		"AttestedCandidateResponse CandidateReceipt=%v",
		r.CandidateReceipt,
	)
}
