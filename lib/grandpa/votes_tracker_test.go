// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"container/list"
	"sort"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/libp2p/go-libp2p/core/peer"
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

func wrapVoteMessageWithPeerID(voteMessage *VoteMessage,
	peerID peer.ID) networkVoteMessage { //nolint:unparam
	return networkVoteMessage{
		from: peerID,
		msg:  voteMessage,
	}
}

func assertVotesMapping(t *testing.T,
	mapping map[common.Hash]map[ed25519.PublicKeyBytes]*list.Element,
	expected map[common.Hash]map[ed25519.PublicKeyBytes]networkVoteMessage) {
	t.Helper()

	require.Len(t, mapping, len(expected), "mapping does not have the expected length")
	for expectedBlockHash, expectedAuthIDToMessage := range expected {
		submap, ok := mapping[expectedBlockHash]
		require.Truef(t, ok, "block hash %s not found in mapping", expectedBlockHash)
		require.Lenf(t, submap, len(expectedAuthIDToMessage),
			"submapping for block hash %s does not have the expected length", expectedBlockHash)
		for expectedAuthorityID, expectedNetworkVoteMessage := range expectedAuthIDToMessage {
			element, ok := submap[expectedAuthorityID]
			assert.Truef(t, ok,
				"submapping for block hash %s does not have expected authority id %s",
				expectedBlockHash, expectedAuthorityID)
			actualNetworkVoteMessage := element.Value.(networkVoteMessage)
			assert.Equalf(t, expectedNetworkVoteMessage, actualNetworkVoteMessage,
				"network vote message for block hash %s and authority id %s is not as expected",
				expectedBlockHash, expectedAuthorityID)
		}
	}
}

func Test_newVotesTracker(t *testing.T) {
	t.Parallel()

	const capacity = 1
	expected := votesTracker{
		mapping:    make(map[common.Hash]map[ed25519.PublicKeyBytes]*list.Element, capacity),
		linkedList: list.New(),
		capacity:   capacity,
	}
	vt := newVotesTracker(capacity)

	assert.Equal(t, expected.mapping, vt.mapping)
	assert.Equal(t, expected.linkedList, vt.linkedList)
	assert.Equal(t, expected.capacity, vt.capacity)
}

// We cannot really unit test each method independently
// due to the dependency on the double linked list from
// the standard package `list` which has private fields
// which cannot be set.
// For example we cannot assert the votes tracker mapping
// entirely due to the linked list elements unexported fields.

func Test_votesTracker_cleanup(t *testing.T) {
	t.Parallel()

	t.Run("in_same_block", func(t *testing.T) {
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
		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]networkVoteMessage{
			blockHashA: {
				authIDB: wrapVoteMessageWithPeerID(messageBlockAAuthB, somePeer),
				authIDC: wrapVoteMessageWithPeerID(messageBlockAAuthC, somePeer),
			},
		})
	})

	t.Run("remove_entire_block", func(t *testing.T) {
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
		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]networkVoteMessage{
			blockHashB: {
				authIDA: wrapVoteMessageWithPeerID(messageBlockBAuthA, somePeer),
				authIDB: wrapVoteMessageWithPeerID(messageBlockBAuthB, somePeer),
			},
		})
	})
}

// This test verifies overidding a value does not affect the
// input order for which each message was added.
func Test_votesTracker_overriding(t *testing.T) {
	t.Parallel()

	t.Run("override_oldest", func(t *testing.T) {
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

		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]networkVoteMessage{
			blockHashB: {
				authIDA: wrapVoteMessageWithPeerID(messageBlockBAuthA, somePeer),
				authIDB: wrapVoteMessageWithPeerID(messageBlockBAuthB, somePeer),
			},
		})
	})

	t.Run("override_newest", func(t *testing.T) {
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

		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]networkVoteMessage{
			blockHashB: {
				authIDA: wrapVoteMessageWithPeerID(messageBlockBAuthA, somePeer),
				authIDB: wrapVoteMessageWithPeerID(messageBlockBAuthB, somePeer),
			},
		})
	})
}

func Test_votesTracker_delete(t *testing.T) {
	t.Parallel()

	t.Run("non_existing_block_hash", func(t *testing.T) {
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

		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]networkVoteMessage{
			blockHashA: {
				authIDA: wrapVoteMessageWithPeerID(messageBlockAAuthA, somePeer),
			},
		})
	})

	t.Run("existing_block_hash", func(t *testing.T) {
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

		assertVotesMapping(t, tracker.mapping, map[common.Hash]map[ed25519.PublicKeyBytes]networkVoteMessage{})
	})
}

func Test_votesTracker_messages(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		votesTracker *votesTracker
		blockHash    common.Hash
		messages     []networkVoteMessage
	}{
		"non_existing_block_hash": {
			votesTracker: &votesTracker{
				mapping: map[common.Hash]map[ed25519.PublicKeyBytes]*list.Element{
					{1}: {},
				},
				linkedList: list.New(),
			},
			blockHash: common.Hash{2},
		},
		"existing_block_hash": {
			votesTracker: &votesTracker{
				mapping: map[common.Hash]map[ed25519.PublicKeyBytes]*list.Element{
					{1}: {
						ed25519.PublicKeyBytes{1}: {
							Value: networkVoteMessage{
								from: "a",
								msg:  &VoteMessage{Round: 1},
							},
						},
						ed25519.PublicKeyBytes{2}: {
							Value: networkVoteMessage{
								from: "a",
								msg:  &VoteMessage{Round: 2},
							},
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
			messages := vt.messages(testCase.blockHash)

			sort.Slice(messages, func(i, j int) bool {
				if messages[i].from == messages[j].from {
					return messages[i].msg.Round < messages[j].msg.Round
				}
				return messages[i].from < messages[j].from
			})

			assert.Equal(t, testCase.messages, messages)
		})
	}
}

func Test_votesTracker_networkVoteMessages(t *testing.T) {
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

	networkVoteMessages := vt.networkVoteMessages()

	expectedNetworkVoteMessages := []networkVoteMessage{
		{from: "a", msg: messageBlockAAuthA},
		{from: "b", msg: messageBlockAAuthB},
		{from: "b", msg: messageBlockBAuthA},
	}

	assert.ElementsMatch(t, expectedNetworkVoteMessages, networkVoteMessages)
}
