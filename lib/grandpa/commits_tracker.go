// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"container/list"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/grandpa/models"
)

// commitsTracker tracks vote messages that could
// not be processed, and removes the oldest ones once
// its maximum capacity is reached.
// It is NOT THREAD SAFE to use.
type commitsTracker struct {
	// map of commit block hash to linked list commit message.
	mapping map[common.Hash]*list.Element
	// double linked list of commit messages
	// to track the order commit messages were added in.
	linkedList *list.List
	capacity   int
}

// newCommitsTracker creates a new commit messages tracker
// with the capacity specified.
func newCommitsTracker(capacity int) commitsTracker {
	return commitsTracker{
		mapping:    make(map[common.Hash]*list.Element, capacity),
		linkedList: list.New(),
		capacity:   capacity,
	}
}

// add adds a commit message to the commit message tracker.
// If the commit message tracker capacity is reached,
// the oldest commit message is removed.
func (ct *commitsTracker) add(commitMessage *models.CommitMessage) {
	blockHash := commitMessage.Vote.Hash

	listElement, has := ct.mapping[blockHash]
	if has {
		// commit already exists so override the commit message in the linked list;
		// do not move the list element in the linked list to avoid
		// someone re-sending the same commit message and going at the
		// front of the list, hence erasing other possible valid commit messages
		// in the tracker.
		listElement.Value = commitMessage
		return
	}

	// add new block hash in tracker
	ct.cleanup()
	listElement = ct.linkedList.PushFront(commitMessage)
	ct.mapping[blockHash] = listElement
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

	oldestCommitMessage := oldestElement.Value.(*models.CommitMessage)
	oldestBlockHash := oldestCommitMessage.Vote.Hash
	delete(ct.mapping, oldestBlockHash)
}

// delete deletes all the vote messages for a particular
// block hash from the vote messages tracker.
func (ct *commitsTracker) delete(blockHash common.Hash) {
	listElement, has := ct.mapping[blockHash]
	if !has {
		return
	}

	ct.linkedList.Remove(listElement)
	delete(ct.mapping, blockHash)
}

// message returns a pointer to the
// commit message for a particular block hash from
// the tracker. It returns nil if the block hash
// does not exist in the tracker
func (ct *commitsTracker) message(blockHash common.Hash) (
	message *models.CommitMessage) {
	listElement, ok := ct.mapping[blockHash]
	if !ok {
		return nil
	}

	return listElement.Value.(*models.CommitMessage)
}

// forEach runs the function `f` on each
// commit message stored in the tracker.
func (ct *commitsTracker) forEach(f func(message *models.CommitMessage)) {
	for _, data := range ct.mapping {
		message := data.Value.(*models.CommitMessage)
		f(message)
	}
}
