// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"golang.org/x/exp/constraints"
)

var (
	VERSION_KEY        = []byte("grandpa_schema_version")
	SET_STATE_KEY      = []byte("grandpa_completed_round")
	CONCLUDED_ROUNDS   = []byte("grandpa_concluded_rounds")
	AUTHORITY_SET_KEY  = []byte("grandpa_voters")
	BEST_JUSTIFICATION = []byte("grandpa_best_justification")
)

const CURRENT_VERSION = uint32(3)

type roundInfo[H comparable, N constraints.Unsigned] struct {
	roundNumber uint64
	roundState  finalityGrandpa.RoundState[H, N]
}

// V1VoterSetState The voter set state.
type V1VoterSetState[H comparable, N constraints.Unsigned] struct {
	isLive    bool // indicates live or paused state
	roundInfo roundInfo[H, N]
}

type V0VoterSetState[H comparable, N constraints.Unsigned] roundInfo[H, N]

type V0PendingChange[H comparable, N constraints.Unsigned] struct {
	NextAuthorities AuthorityList
	Delay           N
	CanonHeight     N
	CanonHash       H
}

type V0AuthoritySet[H comparable, N constraints.Unsigned] struct {
	CurrentAuthorities AuthorityList
	SetID              uint64
	PendingChanges     []V0PendingChange[H, N]
}

type PersistentData[H comparable, N constraints.Unsigned] struct {
	authoritySet AuthoritySet[H, N]
	setState     SharedVoterSetState
}

func (authSet V0AuthoritySet[H, N]) into() AuthoritySet[H, N] {
	pendingStandardChanges := NewChangeTree[H, N]()

	for _, oldChange := range authSet.PendingChanges {
		finalizedKind := Finalized{}
		newChange := PendingChange[H, N]{
			nextAuthorities: oldChange.NextAuthorities,
			delay:           oldChange.Delay,
			canonHeight:     oldChange.CanonHeight,
			canonHash:       oldChange.CanonHash,
			delayKind:       newDelayKind(finalizedKind),
		}

		isDescOf := func(H, H) (bool, error) { return false, nil }
		_, err := pendingStandardChanges.Import(newChange.canonHash, newChange.canonHeight, newChange, isDescOf)
		if err != nil {
			logger.Warnf("migrating pending authority set change: %s", err)
			logger.Warn("node is in a potentially inconsistent state")
		}
	}

	authoritySet := NewAuthoritySet[H, N](authSet.CurrentAuthorities, authSet.SetID, pendingStandardChanges, []PendingChange[H, N]{}, nil)
	if authoritySet == nil {
		panic("current_authorities is non-empty and weights are non-zero; qed.")
	}
	return *authoritySet
}

// Dont decode I think
func loadDecode(client AuxStore, key []byte) *[]byte {
	// Nil case means value not in db, not nil means encoded value retrieved
	return client.GetAux(key)
}

type migrationData[H comparable, N constraints.Unsigned] struct {
	authSet       AuthoritySet[H, N]
	voterSetState VoterSetState
}

func migrateFromVersion0[H comparable, N constraints.Unsigned](client AuxStore, genesisRound func(genesis finalityGrandpa.HashNumber[H, N]) finalityGrandpa.RoundState[H, N], genesisData finalityGrandpa.HashNumber[H, N]) *migrationData[H, N] {
	// TODO
	return nil
}

// Load or initialize persistent data from backend.

type GenesisAuthoritiesFunc func() (AuthorityList, error)

func loadPersistent[H comparable, N constraints.Unsigned](client AuxStore, genesisHash H, genesisNumber N, genesisAuthorities GenesisAuthoritiesFunc) (*PersistentData[H, N], error) {
	encodedVersion := loadDecode(client, VERSION_KEY)

	genesis := finalityGrandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber}
	// I think can just add genesis data as arguments to functions that use this
	makeGenesisRound := finalityGrandpa.NewRoundState[H, N]

	// None case
	if encodedVersion == nil {
		fmt.Println("here")
		migrationInfo := migrateFromVersion0[H, N](client, makeGenesisRound, genesis)
		if migrationInfo != nil {
			return &PersistentData[H, N]{
				authoritySet: migrationInfo.authSet,
				//  TODO fix this cast
				setState: SharedVoterSetState(migrationInfo.voterSetState),
			}, nil
		}
	} else {
		// Handle the some cases

		var version uint32
		err := scale.Unmarshal(*encodedVersion, &version)
		if err != nil {
			return nil, err
		}

		switch version {
		case 1:
			fmt.Println("1")
		case 2:
			fmt.Println("2")
		case 3:
			fmt.Println("3")
		default:
			return nil, fmt.Errorf("unsupported GRANDPA DB version: %v", version)

		}
	}

	return nil, nil
}
