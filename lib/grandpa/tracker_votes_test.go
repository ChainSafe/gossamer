// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"container/list"
	"sort"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildVoteMessage creates a test vote message using the
// given block hash and authority ID only.
func buildVoteMessage(blockHash common.Hash,
	authorityID ed25519.PublicKeyBytes) *VoteMessage {
	return &VoteMessage{
		Message: SignedMessage{
			BlockHash:   blockHash,
			AuthorityID: authorityID,
		},
	}
}

func assertVotesMapping(t *testing.T,
	mapping map[common.Hash]authorityIDToData,
	expected map[common.Hash]map[ed25519.PublicKeyBytes]*VoteMessage) {
	t.Helper()

	require.Len(t, mapping, len(expected), "mapping does not have the expected length")
	for expectedBlockHash, expectedAuthIDToMessage := range expected {
		submap, ok := mapping[expectedBlockHash]
		require.Truef(t, ok, "block hash %s not found in mapping", expectedBlockHash)
		require.Lenf(t, submap, len(expectedAuthIDToMessage),
			"submapping for block hash %s does not have the expected length", expectedBlockHash)
		for expectedAuthorityID, expectedMessage := range expectedAuthIDToMessage {
			data, ok := submap[expectedAuthorityID]
			assert.Truef(t, ok,
				"submapping for block hash %s does not have expected authority id %s",
				expectedBlockHash, expectedAuthorityID)
			assert.Equalf(t, expectedMessage, data.message,
				"message for block hash %s and authority id %s is not as expected",
				expectedBlockHash, expectedAuthorityID)
		}
	}
}

func Test_newVotesTracker(t *testing.T) {
	t.Parallel()

	const capacity = 1
	expected := votesTracker{
		mapping:    make(map[common.Hash]authorityIDToData, capacity),
		linkedList: list.New(),
		capacity:   capacity,
	}
	vt := newVotesTracker(capacity)

	assert.Equal(t, expected, vt)
}

// We cannot really unit test each method independently
// due to the dependency on the double linked list from
// the standard package `list` which has private fields
// which cannot be set.
// For example we cannot assert the votes tracker mapping
// entirely due to the linked list elements unexported fields.

func Test_votesTracker_cleanup(t *testing.T) {
	t.Parallel()

	t.Run("in same block", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newVotesTracker(capacity)

		blockHashA := common.Hash{0xa}

		authIDA := ed25519.PublicKeyBytes{0xa}
		authIDB := ed25519.PublicKeyBytes{0xb}
		authIDC := ed25519.PublicKeyBytes{0xc}

		messageBlockAAuthA := buildVoteMessage(blockHashA, authIDA)
		messageBlockAAuthB := buildVoteMessage(blockHashA, authIDB)
		messageBlockAAuthC := buildVoteMessage(blockHashA, authIDC)

		const somePeer = peer.ID("abc")

		tracker.add(somePeer, messageBlockAAuthA)
		tracker.add(somePeer, messageBlockAAuthB)
		// Add third message for block A and authority id C.
		// This triggers a cleanup removing the oldest message
		// which is for block A and authority id A.
		tracker.add(somePeer, messageBlockAAuthC)
		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]*VoteMessage{
			blockHashA: {
				authIDB: messageBlockAAuthB,
				authIDC: messageBlockAAuthC,
			},
		})
	})

	t.Run("remove entire block", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newVotesTracker(capacity)

		blockHashA := common.Hash{0xa}
		blockHashB := common.Hash{0xb}

		authIDA := ed25519.PublicKeyBytes{0xa}
		authIDB := ed25519.PublicKeyBytes{0xb}

		messageBlockAAuthA := buildVoteMessage(blockHashA, authIDA)
		messageBlockBAuthA := buildVoteMessage(blockHashB, authIDA)
		messageBlockBAuthB := buildVoteMessage(blockHashB, authIDB)

		const somePeer = peer.ID("abc")

		tracker.add(somePeer, messageBlockAAuthA)
		tracker.add(somePeer, messageBlockBAuthA)
		// Add third message for block B and authority id B.
		// This triggers a cleanup removing the oldest message
		// which is for block A and authority id A. The block A
		// is also completely removed since it does not contain
		// any authority ID (vote message) anymore.
		tracker.add(somePeer, messageBlockBAuthB)
		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]*VoteMessage{
			blockHashB: {
				authIDA: messageBlockBAuthA,
				authIDB: messageBlockBAuthB,
			},
		})
	})
}

// This test verifies overidding a value does not affect the
// input order for which each message was added.
func Test_votesTracker_overriding(t *testing.T) {
	t.Parallel()

	t.Run("override oldest", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newVotesTracker(capacity)

		blockHashA := common.Hash{0xa}
		blockHashB := common.Hash{0xb}

		authIDA := ed25519.PublicKeyBytes{0xa}
		authIDB := ed25519.PublicKeyBytes{0xb}

		messageBlockAAuthA := buildVoteMessage(blockHashA, authIDA)
		messageBlockBAuthA := buildVoteMessage(blockHashB, authIDA)
		messageBlockBAuthB := buildVoteMessage(blockHashB, authIDB)

		const somePeer = peer.ID("abc")

		tracker.add(somePeer, messageBlockAAuthA)
		tracker.add(somePeer, messageBlockBAuthA)
		tracker.add(somePeer, messageBlockAAuthA) // override oldest
		tracker.add(somePeer, messageBlockBAuthB)

		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]*VoteMessage{
			blockHashB: {
				authIDA: messageBlockBAuthA,
				authIDB: messageBlockBAuthB,
			},
		})
	})

	t.Run("override newest", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newVotesTracker(capacity)

		blockHashA := common.Hash{0xa}
		blockHashB := common.Hash{0xb}

		authIDA := ed25519.PublicKeyBytes{0xa}
		authIDB := ed25519.PublicKeyBytes{0xb}

		messageBlockAAuthA := buildVoteMessage(blockHashA, authIDA)
		messageBlockBAuthA := buildVoteMessage(blockHashB, authIDA)
		messageBlockBAuthB := buildVoteMessage(blockHashB, authIDB)

		const somePeer = peer.ID("abc")

		tracker.add(somePeer, messageBlockAAuthA)
		tracker.add(somePeer, messageBlockBAuthA)
		tracker.add(somePeer, messageBlockBAuthA) // override newest
		tracker.add(somePeer, messageBlockBAuthB)

		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]*VoteMessage{
			blockHashB: {
				authIDA: messageBlockBAuthA,
				authIDB: messageBlockBAuthB,
			},
		})
	})
}

func Test_votesTracker_delete(t *testing.T) {
	t.Parallel()

	t.Run("non existing block hash", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newVotesTracker(capacity)

		blockHashA := common.Hash{0xa}
		blockHashB := common.Hash{0xb}

		authIDA := ed25519.PublicKeyBytes{0xa}

		messageBlockAAuthA := buildVoteMessage(blockHashA, authIDA)

		const somePeer = peer.ID("abc")

		tracker.add(somePeer, messageBlockAAuthA)
		tracker.delete(blockHashB)

		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]*VoteMessage{
			blockHashA: {
				authIDA: messageBlockAAuthA,
			},
		})
	})

	t.Run("existing block hash", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newVotesTracker(capacity)

		blockHashA := common.Hash{0xa}
		authIDA := ed25519.PublicKeyBytes{0xa}
		authIDB := ed25519.PublicKeyBytes{0xb}
		messageBlockAAuthA := buildVoteMessage(blockHashA, authIDA)
		messageBlockAAuthB := buildVoteMessage(blockHashA, authIDB)

		const somePeer = peer.ID("abc")

		tracker.add(somePeer, messageBlockAAuthA)
		tracker.add(somePeer, messageBlockAAuthB)
		tracker.delete(blockHashA)

		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]*VoteMessage{})
	})
}

func Test_votesTracker_getMessagesForBlockHash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		votesTracker *votesTracker
		blockHash    common.Hash
		messages     []networkVoteMessage
	}{
		"non existing block hash": {
			votesTracker: &votesTracker{
				mapping: map[common.Hash]authorityIDToData{
					{1}: {},
				},
				linkedList: list.New(),
			},
			blockHash: common.Hash{2},
		},
		"existing block hash": {
			votesTracker: &votesTracker{
				mapping: map[common.Hash]authorityIDToData{
					{1}: {
						ed25519.PublicKeyBytes{1}: {
							peerID:  "a",
							message: &VoteMessage{Round: 1},
						},
						ed25519.PublicKeyBytes{2}: {
							peerID:  "a",
							message: &VoteMessage{Round: 2},
						},
					},
				},
			},
			blockHash: common.Hash{1},
			messages: []networkVoteMessage{
				{from: peer.ID("a"), msg: &VoteMessage{Round: 1}},
				{from: peer.ID("a"), msg: &VoteMessage{Round: 2}},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			vt := testCase.votesTracker
			messages := vt.getMessagesForBlockHash(testCase.blockHash)

			assert.Equal(t, testCase.messages, messages)
		})
	}
}

func Test_votesTracker_forEach(t *testing.T) {
	t.Parallel()

	const capacity = 10
	vt := newVotesTracker(capacity)

	blockHashA := common.Hash{0xa}
	blockHashB := common.Hash{0xb}

	authIDA := ed25519.PublicKeyBytes{0xa}
	authIDB := ed25519.PublicKeyBytes{0xb}

	messageBlockAAuthA := buildVoteMessage(blockHashA, authIDA)
	messageBlockAAuthB := buildVoteMessage(blockHashA, authIDB)
	messageBlockBAuthA := buildVoteMessage(blockHashB, authIDA)

	vt.add("a", messageBlockAAuthA)
	vt.add("b", messageBlockAAuthB)
	vt.add("b", messageBlockBAuthA)

	type result struct {
		peerID  peer.ID
		message *VoteMessage
	}
	var results []result

	vt.forEach(func(peerID peer.ID, message *VoteMessage) {
		results = append(results, result{
			peerID:  peerID,
			message: message,
		})
	})

	// Predictable messages order for assertion.
	// Sort by block hash then authority id then peer ID.
	sort.Slice(results, func(i, j int) bool {
		blockHashFirst := results[i].message.Message.BlockHash
		blockHashSecond := results[j].message.Message.BlockHash
		if blockHashFirst == blockHashSecond {
			authIDFirst := results[i].message.Message.AuthorityID
			authIDSecond := results[j].message.Message.AuthorityID
			if authIDFirst == authIDSecond {
				return results[i].peerID < results[j].peerID
			}
			return bytes.Compare(authIDFirst[:], authIDSecond[:]) < 0
		}
		return bytes.Compare(blockHashFirst[:], blockHashSecond[:]) < 0
	})

	expectedResults := []result{
		{peerID: "a", message: messageBlockAAuthA},
		{peerID: "b", message: messageBlockAAuthB},
		{peerID: "b", message: messageBlockBAuthA},
	}

	assert.Equal(t, expectedResults, results)
}
