// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

type AuthorityID any

type AuthoritySignature any

// Authority represents a grandpa authority
type Authority[ID AuthorityID] struct {
	Key    ID
	Weight uint64
}

type AuthorityList[ID AuthorityID] []Authority[ID]

// NewAuthoritySetStruct A new authority set along with the canonical block it changed at.
type NewAuthoritySetStruct[H comparable, N constraints.Unsigned, ID AuthorityID] struct {
	CanonNumber N
	CanonHash   H
	SetId       N
	Authorities []Authority[ID]
}

type messageData[H comparable, N constraints.Unsigned] struct {
	Round   uint64
	SetID   uint64
	Message finalityGrandpa.Message[H, N]
}

// Check a message signature by encoding the message as a localised payload and
// verifying the provided signature using the expected authority id.
// The encoding necessary to verify the signature will be done using the given
// buffer, the original content of the buffer will be cleared.
func checkMessageSignature[H comparable, N constraints.Unsigned](
	message any,
	id any,
	signature any,
	round uint64,
	setID uint64) (bool, error) {

	msg, ok := message.(finalityGrandpa.Message[H, N])
	if !ok {
		return false, fmt.Errorf("invalid cast to finalityGrandpa.Message[H, N]")
	}

	castedID, ok := id.(*ed25519.PublicKey)
	if !ok {
		return false, fmt.Errorf("invalid cast to *ed25519.PublicKey")
	}

	sig, ok := signature.([]byte)
	if !ok {
		return false, fmt.Errorf("invalid cast to []byte")
	}

	m := messageData[H, N]{
		round,
		setID,
		msg,
	}

	enc, err := scale.Marshal(m)
	if err != nil {
		return false, err
	}
	valid, err := castedID.Verify(enc, sig[:])
	if err != nil {
		return false, err
	}
	return valid, nil
}
