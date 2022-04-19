// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"bytes"
	"container/list"
	"crypto/rand"
	"sort"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildCommitMessage creates a test commit message
// using the given block hash.
func buildCommitMessage(blockHash common.Hash) *CommitMessage {
	return &CommitMessage{
		Vote: Vote{
			Hash: blockHash,
		},
	}
}

func assertCommitsMapping(t *testing.T,
	mapping map[common.Hash]commitMessageMapData,
	expected map[common.Hash]*CommitMessage) {
	t.Helper()

	require.Len(t, mapping, len(expected), "mapping does not have the expected length")
	for expectedBlockHash, expectedCommitMessage := range expected {
		data, ok := mapping[expectedBlockHash]
		assert.Truef(t, ok, "block hash %s not found in mapping", expectedBlockHash)
		assert.Equalf(t, expectedCommitMessage, data.message,
			"commit message for block hash %s is not as expected",
			expectedBlockHash)
	}
}

func Test_newCommitsTracker(t *testing.T) {
	t.Parallel()

	const capacity = 1
	expected := commitsTracker{
		mapping:    make(map[common.Hash]commitMessageMapData, capacity),
		linkedList: list.New(),
		capacity:   capacity,
	}
	vt := newCommitsTracker(capacity)

	assert.Equal(t, expected, vt)
}

// We cannot really unit test each method independently
// due to the dependency on the double linked list from
// the standard package `list` which has private fields
// which cannot be set.
// For example we cannot assert the commits tracker mapping
// entirely due to the linked list elements unexported fields.

func Test_commitsTracker_cleanup(t *testing.T) {
	t.Parallel()

	const capacity = 2
	tracker := newCommitsTracker(capacity)

	blockHashA := common.Hash{0xa}
	blockHashB := common.Hash{0xb}
	blockHashC := common.Hash{0xc}

	messageBlockA := buildCommitMessage(blockHashA)
	messageBlockB := buildCommitMessage(blockHashB)
	messageBlockC := buildCommitMessage(blockHashC)

	tracker.add(messageBlockA)
	tracker.add(messageBlockB)
	// Add third message for block C.
	// This triggers a cleanup removing the oldest message
	// which is the message for block A.
	tracker.add(messageBlockC)
	assertCommitsMapping(t, tracker.mapping, map[common.Hash]*CommitMessage{
		blockHashB: messageBlockB,
		blockHashC: messageBlockC,
	})
}

// This test verifies overidding a value does not affect the
// input order for which each message was added.
func Test_commitsTracker_overriding(t *testing.T) {
	t.Parallel()

	t.Run("override oldest", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newCommitsTracker(capacity)

		blockHashA := common.Hash{0xa}
		blockHashB := common.Hash{0xb}
		blockHashC := common.Hash{0xc}

		messageBlockA := buildCommitMessage(blockHashA)
		messageBlockB := buildCommitMessage(blockHashB)
		messageBlockC := buildCommitMessage(blockHashC)

		tracker.add(messageBlockA)
		tracker.add(messageBlockB)
		tracker.add(messageBlockA) // override oldest
		tracker.add(messageBlockC)

		assertCommitsMapping(t, tracker.mapping, map[common.Hash]*CommitMessage{
			blockHashB: messageBlockB,
			blockHashC: messageBlockC,
		})
	})

	t.Run("override newest", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newCommitsTracker(capacity)

		blockHashA := common.Hash{0xa}
		blockHashB := common.Hash{0xb}
		blockHashC := common.Hash{0xc}

		messageBlockA := buildCommitMessage(blockHashA)
		messageBlockB := buildCommitMessage(blockHashB)
		messageBlockC := buildCommitMessage(blockHashC)

		tracker.add(messageBlockA)
		tracker.add(messageBlockB)
		tracker.add(messageBlockB) // override newest
		tracker.add(messageBlockC)

		assertCommitsMapping(t, tracker.mapping, map[common.Hash]*CommitMessage{
			blockHashB: messageBlockB,
			blockHashC: messageBlockC,
		})
	})
}

func Test_commitsTracker_delete(t *testing.T) {
	t.Parallel()

	t.Run("non existing block hash", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newCommitsTracker(capacity)

		blockHashA := common.Hash{0xa}
		blockHashB := common.Hash{0xb}

		messageBlockA := buildCommitMessage(blockHashA)

		tracker.add(messageBlockA)
		tracker.delete(blockHashB)

		assertCommitsMapping(t, tracker.mapping, map[common.Hash]*CommitMessage{
			blockHashA: messageBlockA,
		})
	})

	t.Run("existing block hash", func(t *testing.T) {
		t.Parallel()

		const capacity = 2
		tracker := newCommitsTracker(capacity)

		blockHashA := common.Hash{0xa}
		blockHashB := common.Hash{0xb}

		messageBlockA := buildCommitMessage(blockHashA)
		messageBlockB := buildCommitMessage(blockHashB)

		tracker.add(messageBlockA)
		tracker.add(messageBlockB)
		tracker.delete(blockHashB)

		assertCommitsMapping(t, tracker.mapping, map[common.Hash]*CommitMessage{
			blockHashA: messageBlockA,
		})
	})
}

func Test_commitsTracker_getMessagesForBlockHash(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		commitsTracker *commitsTracker
		blockHash      common.Hash
		message        *CommitMessage
	}{
		"non existing block hash": {
			commitsTracker: &commitsTracker{
				mapping: map[common.Hash]commitMessageMapData{
					{1}: {},
				},
			},
			blockHash: common.Hash{2},
		},
		"existing block hash": {
			commitsTracker: &commitsTracker{
				mapping: map[common.Hash]commitMessageMapData{
					{1}: {
						message: &CommitMessage{Round: 1},
					},
				},
			},
			blockHash: common.Hash{1},
			message:   &CommitMessage{Round: 1},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			vt := testCase.commitsTracker
			message := vt.getMessageForBlockHash(testCase.blockHash)

			assert.Equal(t, testCase.message, message)
		})
	}
}

func Test_commitsTracker_forEach(t *testing.T) {
	t.Parallel()

	const capacity = 10
	ct := newCommitsTracker(capacity)

	blockHashA := common.Hash{0xa}
	blockHashB := common.Hash{0xb}
	blockHashC := common.Hash{0xc}

	messageBlockA := buildCommitMessage(blockHashA)
	messageBlockB := buildCommitMessage(blockHashB)
	messageBlockC := buildCommitMessage(blockHashC)

	ct.add(messageBlockA)
	ct.add(messageBlockB)
	ct.add(messageBlockC)

	var results []*CommitMessage
	ct.forEach(func(message *CommitMessage) {
		results = append(results, message)
	})

	// Predictable messages order for assertion.
	// Sort by block hash then authority id then peer ID.
	sort.Slice(results, func(i, j int) bool {
		return bytes.Compare(results[i].Vote.Hash[:],
			results[j].Vote.Hash[:]) < 0
	})

	expectedResults := []*CommitMessage{
		messageBlockA,
		messageBlockB,
		messageBlockC,
	}

	assert.Equal(t, expectedResults, results)
}

func Benchmark_ForEachVsSlice(b *testing.B) {
	getMessages := func(ct *commitsTracker) (messages []*CommitMessage) {
		messages = make([]*CommitMessage, 0, len(ct.mapping))
		for _, data := range ct.mapping {
			messages = append(messages, data.message)
		}
		return messages
	}

	f := func(message *CommitMessage) {
		message.Round++
		message.SetID++
	}

	const trackerSize = 10e4
	makeSeededTracker := func() (ct *commitsTracker) {
		ct = &commitsTracker{
			mapping: make(map[common.Hash]commitMessageMapData),
		}
		for i := 0; i < trackerSize; i++ {
			hashBytes := make([]byte, 32)
			_, _ = rand.Read(hashBytes)
			var blockHash common.Hash
			copy(blockHash[:], hashBytes)
			ct.mapping[blockHash] = commitMessageMapData{
				message: &CommitMessage{
					Round: uint64(i),
					SetID: uint64(i),
				},
			}
		}
		return ct
	}

	b.Run("forEach", func(b *testing.B) {
		tracker := makeSeededTracker()
		for i := 0; i < b.N; i++ {
			tracker.forEach(f)
		}
	})

	b.Run("get messages for iterate", func(b *testing.B) {
		tracker := makeSeededTracker()
		for i := 0; i < b.N; i++ {
			messages := getMessages(tracker)
			for _, message := range messages {
				f(message)
			}
		}
	})
}
