// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"container/list"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/libp2p/go-libp2p-core/peer"
)

// votesTracker tracks vote messages that could
// not be processed, and removes the oldest ones once
// its maximum capacity is reached.
// It is NOT THREAD SAFE to use.
type votesTracker struct {
	// map of vote block hash to authority ID (ed25519 public Key)
	// to linked list element pointer
	mapping map[common.Hash]map[ed25519.PublicKeyBytes]*list.Element
	// double linked list of voteMessageData (peer ID + Vote Message)
	linkedList *list.List
	capacity   int
}

// newVotesTracker creates a new vote message tracker
// with the capacity specified.
func newVotesTracker(capacity int) votesTracker {
	return votesTracker{
		mapping:    make(map[common.Hash]map[ed25519.PublicKeyBytes]*list.Element, capacity),
		linkedList: list.New(),
		capacity:   capacity,
	}
}

// add adds a vote message to the vote message tracker.
// If the vote message tracker capacity is reached,
// the oldest vote message is removed.
func (vt *votesTracker) add(peerID peer.ID, voteMessage *VoteMessage) {
	signedMessage := voteMessage.Message
	blockHash := signedMessage.BlockHash
	authorityID := signedMessage.AuthorityID

	authorityIDToElement, blockHashExists := vt.mapping[blockHash]
	if blockHashExists {
		element, voteExists := authorityIDToElement[authorityID]
		if voteExists {
			// vote already exists so override the vote for the authority ID;
			// do not move the list element in the linked list to avoid
			// someone re-sending an equivocatory vote message and going at the
			// front of the list, hence erasing other possible valid vote messages
			// in the tracker.
			element.Value = networkVoteMessage{
				from: peerID,
				msg:  voteMessage,
			}
			return
		}
		// continue below and add the authority ID and data to the tracker.
	} else {
		// add new block hash in tracker
		authorityIDToElement = make(map[ed25519.PublicKeyBytes]*list.Element)
		vt.mapping[blockHash] = authorityIDToElement
		// continue below and add the authority ID and data to the tracker.
	}

	vt.cleanup()
	elementData := networkVoteMessage{
		from: peerID,
		msg:  voteMessage,
	}
	element := vt.linkedList.PushFront(elementData)
	authorityIDToElement[authorityID] = element
}

// cleanup removes the oldest vote message from the tracker
// if the number of vote messages is at the tracker capacity.
// This method is designed to be called automatically from the
// add method and should not be called elsewhere.
func (vt *votesTracker) cleanup() {
	if vt.linkedList.Len() < vt.capacity {
		return
	}

	oldestElement := vt.linkedList.Back()
	vt.linkedList.Remove(oldestElement)

	oldestData := oldestElement.Value.(networkVoteMessage)
	oldestBlockHash := oldestData.msg.Message.BlockHash
	oldestAuthorityID := oldestData.msg.Message.AuthorityID

	authIDToElement := vt.mapping[oldestBlockHash]

	delete(authIDToElement, oldestAuthorityID)
	if len(authIDToElement) == 0 {
		delete(vt.mapping, oldestBlockHash)
	}
}

// delete deletes all the vote messages for a particular
// block hash from the vote messages tracker.
func (vt *votesTracker) delete(blockHash common.Hash) {
	authIDToElement, has := vt.mapping[blockHash]
	if !has {
		return
	}

	for _, element := range authIDToElement {
		vt.linkedList.Remove(element)
	}

	delete(vt.mapping, blockHash)
}

// messages returns all the vote messages
// for a particular block hash from the tracker as a slice
// of networkVoteMessage. There is no order in the slice.
// It returns nil if the block hash does not exist.
func (vt *votesTracker) messages(blockHash common.Hash) (
	messages []networkVoteMessage) {
	authIDToElement, ok := vt.mapping[blockHash]
	if !ok {
		// Note authIDToElement cannot be empty
		return nil
	}

	messages = make([]networkVoteMessage, 0, len(authIDToElement))
	for _, element := range authIDToElement {
		message := element.Value.(networkVoteMessage)
		messages = append(messages, message)
	}
	return messages
}

// networkVoteMessages returns all pairs of
// peer id + message stored in the tracker
// as a slice of networkVoteMessages.
func (vt *votesTracker) networkVoteMessages() (
	messages []networkVoteMessage) {
	messages = make([]networkVoteMessage, 0, vt.linkedList.Len())
	for _, authorityIDToElement := range vt.mapping {
		for _, element := range authorityIDToElement {
			message := element.Value.(networkVoteMessage)
			messages = append(messages, message)
		}
	}
	return messages
}
