// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"golang.org/x/exp/constraints"
)

var logger = log.NewFromGlobal(log.AddContext("consensus", "grandpa"))

// Authority represents a grandpa authority
type Authority struct {
	Key    ed25519.PublicKey
	Weight uint64
}

// NewAuthoritySetStruct A new authority set along with the canonical block it changed at.
type NewAuthoritySetStruct[H comparable, N constraints.Unsigned] struct {
	CanonNumber N
	CanonHash   H
	SetId       N
	Authorities []Authority
}

type ClientForGrandpa interface{}

type Backend interface{}

type Config struct {
	GossipDuration time.Duration
}

type VoterWork[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered] struct {
	voter            *grandpa.Voter[Hash, Number, Signature, ID]
	sharedVoterState any
	env              any
	voterCommandsRx  any
	network          any
	telemetry        any
	metrics          any
}

func NewVoterWork[Hash constraints.Ordered, Number constraints.Unsigned, Signature comparable, ID constraints.Ordered](
	client ClientForGrandpa,
	config Config,

) *VoterWork[Hash, Number, Signature, ID] {
	// grandpa.NewVoter[]()
	return nil
}
