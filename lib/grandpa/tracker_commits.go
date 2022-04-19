// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"container/list"

	"github.com/ChainSafe/gossamer/lib/common"
)

// commitsTracker tracks vote messages that could
// not be processed, and removes the oldest ones once
// its maximum capacity is reached.
// It is NOT THREAD SAFE to use.
type commitsTracker struct {
	// map of commit block hash to data
	// data = message + tracking linked list element pointer
	mapping map[common.Hash]commitMessageMapData
	// double linked list of block hash
	// to track the order commit messages were added in.
	linkedList *list.List
	capacity   int
}

type commitMessageMapData struct {
	message *CommitMessage
	// element contains a block hash value.
	element *list.Element
}

// newCommitsTracker creates a new commit messages tracker
// with the capacity specified.
func newCommitsTracker(capacity int) commitsTracker {
	return commitsTracker{
		mapping:    make(map[common.Hash]commitMessageMapData, capacity),
		linkedList: list.New(),
		capacity:   capacity,
	}
}

// add adds a commit message to the commit message tracker.
// If the commit message tracker capacity is reached,
// the oldest commit message is removed.
func (ct *commitsTracker) add(commitMessage *CommitMessage) {
	blockHash := commitMessage.Vote.Hash

	data, has := ct.mapping[blockHash]
	if has {
		// commit already exists so override the commit for the block hash;
		// do not move the list element in the linked list to avoid
		// someone re-sending the same commit message and going at the
		// front of the list, hence erasing other possible valid commit messages
		// in the tracker.
		data.message = commitMessage
		ct.mapping[blockHash] = data
		return
	}

	// add new block hash in tracker
	ct.cleanup()
	element := ct.linkedList.PushFront(blockHash)
	data = commitMessageMapData{
		message: commitMessage,
		element: element,
	}
	ct.mapping[blockHash] = data
}

// cleanup removes the oldest commit message from the tracker
// if the number of commit messages is at the tracker capacity.
// This method is designed to be called automatically from the
// add method and should not be called elsewhere.
func (ct *commitsTracker) cleanup() {
	if ct.linkedList.Len() < ct.capacity {
		return
	}

	oldestElement := ct.linkedList.Back()
	ct.linkedList.Remove(oldestElement)

	oldestBlockHash := oldestElement.Value.(common.Hash)
	delete(ct.mapping, oldestBlockHash)
}

// delete deletes all the vote messages for a particular
// block hash from the vote messages tracker.
func (ct *commitsTracker) delete(blockHash common.Hash) {
	data, has := ct.mapping[blockHash]
	if !has {
		return
	}

	ct.linkedList.Remove(data.element)
	delete(ct.mapping, blockHash)
}

// getMessageForBlockHash returns a pointer to the
// commit message for a particular block hash from
// the tracker. It returns nil if the block hash
// does not exist in the tracker
func (ct *commitsTracker) getMessageForBlockHash(
	blockHash common.Hash) (message *CommitMessage) {
	data, ok := ct.mapping[blockHash]
	if !ok {
		return nil
	}

	return data.message
}

// forEach runs the function `f` on each
// commit message stored in the tracker.
func (ct *commitsTracker) forEach(f func(message *CommitMessage)) {
	for _, data := range ct.mapping {
		f(data.message)
	}
}
