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

func migrateFromVersion0[H comparable, N constraints.Unsigned](client AuxStore, genesisRound func(genesis finalityGrandpa.HashNumber[H, N]) finalityGrandpa.RoundState[H, N], genesisData finalityGrandpa.HashNumber[H, N]) (*migrationData[H, N], error) {
	// I think I need to encode current version

	// Maybe this should return error?
	insert := map[string][]byte{}
	insert[string(VERSION_KEY)] = scale.MustMarshal(CURRENT_VERSION)
	err := client.InsertAux(insert, nil)
	if err != nil {
		return nil, err
	}

	encAuthSet := loadDecode(client, AUTHORITY_SET_KEY)
	if encAuthSet != nil {
		var oldSet V0AuthoritySet[H, N]
		fmt.Println(*encAuthSet)
		err = scale.Unmarshal(*encAuthSet, &oldSet)
		if err != nil {
			return nil, err
		}
		newSet := oldSet.into()

		insert = map[string][]byte{}
		insert[string(AUTHORITY_SET_KEY)] = scale.MustMarshal(newSet)
		err = client.InsertAux(insert, nil)
		if err != nil {
			return nil, err
		}

		// Get last voter set state
		var lastRoundVoterSetState V0VoterSetState[H, N]
		encSetState := loadDecode(client, SET_STATE_KEY)
		if encSetState != nil {
			err = scale.Unmarshal(*encSetState, &lastRoundVoterSetState)
			if err != nil {
				return nil, err
			}
		} else {
			lastRoundVoterSetState = V0VoterSetState[H, N]{
				0, genesisRound(genesisData),
			}
		}

		_ = newSet.setId
		base := lastRoundVoterSetState.roundState.PrevoteGHOST
		if base == nil {
			panic("state is for completed round; completed rounds must have a prevote ghost; qed.")
		}

		curRounds := NewCurrentRounds()
		curRounds.Set(lastRoundVoterSetState.roundNumber+1, HasVoted{}) // Needs to be no case

		// TODO set setState as Live with updated info
		setState := VoterSetState{}

		insert = map[string][]byte{}
		insert[string(SET_STATE_KEY)] = scale.MustMarshal(setState)
		err = client.InsertAux(insert, nil)
		if err != nil {
			return nil, err
		}
		return &migrationData[H, N]{
			authSet:       newSet,
			voterSetState: setState,
		}, nil
	}

	return nil, nil
}

func migrateFromVersion1[H comparable, N constraints.Unsigned](client AuxStore, genesisRound func(genesis finalityGrandpa.HashNumber[H, N]) finalityGrandpa.RoundState[H, N], genesisData finalityGrandpa.HashNumber[H, N]) *migrationData[H, N] {
	// TODO
	return nil
}

func migrateFromVersion2[H comparable, N constraints.Unsigned](client AuxStore, genesisRound func(genesis finalityGrandpa.HashNumber[H, N]) finalityGrandpa.RoundState[H, N], genesisData finalityGrandpa.HashNumber[H, N]) *migrationData[H, N] {
	// TODO
	return nil
}

type GenesisAuthoritiesFunc func() (AuthorityList, error)

// Load or initialize persistent data from backend.
func loadPersistent[H comparable, N constraints.Unsigned](client AuxStore, genesisHash H, genesisNumber N, genesisAuthorities GenesisAuthoritiesFunc) (*PersistentData[H, N], error) {
	encodedVersion := loadDecode(client, VERSION_KEY)

	genesis := finalityGrandpa.HashNumber[H, N]{Hash: genesisHash, Number: genesisNumber}
	// I think can just add genesis data as arguments to functions that use this
	makeGenesisRound := finalityGrandpa.NewRoundState[H, N]

	// None case
	if encodedVersion == nil {
		fmt.Println("none case")
		migrationInfo, err := migrateFromVersion0[H, N](client, makeGenesisRound, genesis)
		if err != nil {
			return nil, err
		}
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
			migrationInfo := migrateFromVersion1[H, N](client, makeGenesisRound, genesis)
			if migrationInfo != nil {
				return &PersistentData[H, N]{
					authoritySet: migrationInfo.authSet,
					//  TODO fix this cast
					setState: SharedVoterSetState(migrationInfo.voterSetState),
				}, nil
			}
		case 2:
			fmt.Println("2")
			migrationInfo := migrateFromVersion2[H, N](client, makeGenesisRound, genesis)
			if migrationInfo != nil {
				return &PersistentData[H, N]{
					authoritySet: migrationInfo.authSet,
					//  TODO fix this cast
					setState: SharedVoterSetState(migrationInfo.voterSetState),
				}, nil
			}
		case 3:
			fmt.Println("3")
			encodedAuthSet := loadDecode(client, AUTHORITY_SET_KEY)
			if encodedAuthSet != nil {
				var setState VoterSetState
				encodedSetState := loadDecode(client, SET_STATE_KEY)
				if encodedSetState != nil {
					// Some case
					err = scale.Unmarshal(*encodedSetState, &setState)
					if err != nil {
						return nil, err
					}
				} else {
					// None case
					state := makeGenesisRound(genesis)
					base := state.PrevoteGHOST
					if base != nil {
						// VoterSetState as live
					} else {
						panic("state is for completed round; completed rounds must have a prevote ghost; qed")
					}
				}
				var set AuthoritySet[H, N]
				err = scale.Unmarshal(*encodedAuthSet, &set)
				if err != nil {
					return nil, err
				}
				return &PersistentData[H, N]{
					// Fix cast
					authoritySet: set,
					// Fix cast
					setState: SharedVoterSetState(setState),
				}, nil
			}
		default: // Some(other), is this same as default? Think not
			return nil, fmt.Errorf("unsupported GRANDPA DB version: %v", version)

		}
	}

	// TODO investigate how load can lead to this case
	logger.Infof("👴 Loading GRANDPA authority set from genesis on what appears to be first startup.")

	genesisAuths, err := genesisAuthorities()
	if err != nil {
		return nil, err
	}
	genesisSet := NewGenesisAuthoritySet[H, N](genesisAuths)
	if genesisSet == nil {
		panic("genesis authorities is non-empty; all weights are non-zero; qed.")
	}

	state := makeGenesisRound(genesis)
	base := state.PrevoteGHOST
	if base == nil {
		panic("state is for completed round; completed rounds must have a prevote ghost; qed.")
	}
	// This should be VoterSetState::Live
	genesisState := VoterSetState{}

	insert := map[string][]byte{}
	insert[string(AUTHORITY_SET_KEY)] = scale.MustMarshal(*genesisSet)
	insert[string(SET_STATE_KEY)] = scale.MustMarshal(genesisState)

	err = client.InsertAux(insert, nil)
	if err != nil {
		return nil, err
	}

	return &PersistentData[H, N]{
		// Fix cast
		authoritySet: *genesisSet,
		// Fix cast
		setState: SharedVoterSetState(genesisState),
	}, nil
}
