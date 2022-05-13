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
	// to data (peer id + message + tracking linked list element pointer)
	mapping map[common.Hash]authorityIDToData
	// double linked list of block hash + authority ID
	// to track the order vote messages were added in.
	linkedList *list.List
	capacity   int
}

type authorityIDToData map[ed25519.PublicKeyBytes]voteMessageMapData

type voteMessageMapData struct {
	peerID  peer.ID
	message *VoteMessage
	// element contains a blockHashAuthID value which
	// itself contains a block hash and an authority ID.
	element *list.Element
}

type blockHashAuthID struct {
	blockHash   common.Hash
	authorityID ed25519.PublicKeyBytes
}

// newVotesTracker creates a new vote message tracker
// with the capacity specified.
func newVotesTracker(capacity int) votesTracker {
	return votesTracker{
		mapping:    make(map[common.Hash]authorityIDToData, capacity),
		linkedList: list.New(),
		capacity:   capacity,
	}
}

func newBlockHashAuthID(blockHash common.Hash,
	authorityID ed25519.PublicKeyBytes) blockHashAuthID {
	return blockHashAuthID{
		blockHash:   blockHash,
		authorityID: authorityID,
	}
}

// add adds a vote message to the vote message tracker.
// If the vote message tracker capacity is reached,
// the oldest vote message is removed.
func (vt *votesTracker) add(peerID peer.ID, voteMessage *VoteMessage) {
	signedMessage := voteMessage.Message
	blockHash := signedMessage.BlockHash
	authorityID := signedMessage.AuthorityID

	voteMessages, blockHashExists := vt.mapping[blockHash]
	if blockHashExists {
		data, voteExists := voteMessages[authorityID]
		if voteExists {
			// vote already exists so override the vote for the authority ID;
			// do not move the list element in the linked list to avoid
			// someone re-sending an equivocatory vote message and going at the
			// front of the list, hence erasing other possible valid vote messages
			// in the tracker.
			data.peerID = peerID
			data.message = voteMessage
			voteMessages[authorityID] = data
			return
		}
		// continue below and add the authority ID and data to the tracker.
	} else {
		// add new block hash in tracker
		voteMessages = make(authorityIDToData)
		vt.mapping[blockHash] = voteMessages
		// continue below and add the authority ID and data to the tracker.
	}

	vt.cleanup()
	elementData := newBlockHashAuthID(blockHash, authorityID)
	element := vt.linkedList.PushFront(elementData)
	data := voteMessageMapData{
		peerID:  peerID,
		message: voteMessage,
		element: element,
	}
	voteMessages[authorityID] = data
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

	oldestData := oldestElement.Value.(blockHashAuthID)
	authIDToData := vt.mapping[oldestData.blockHash]

	delete(authIDToData, oldestData.authorityID)
	if len(authIDToData) == 0 {
		delete(vt.mapping, oldestData.blockHash)
	}
}

// delete deletes all the vote messages for a particular
// block hash from the vote messages tracker.
func (vt *votesTracker) delete(blockHash common.Hash) {
	authIDToData, has := vt.mapping[blockHash]
	if !has {
		return
	}

	for _, data := range authIDToData {
		vt.linkedList.Remove(data.element)
	}

	delete(vt.mapping, blockHash)
}

// messages returns all the vote messages
// for a particular block hash from the tracker as a slice
// of networkVoteMessage. There is no order in the slice.
// It returns nil if the block hash does not exist.
func (vt *votesTracker) messages(blockHash common.Hash) (
	messages []networkVoteMessage) {
	authIDToData, ok := vt.mapping[blockHash]
	if !ok {
		// Note authIDToData cannot be empty
		return nil
	}

	messages = make([]networkVoteMessage, 0, len(authIDToData))
	for _, data := range authIDToData {
		message := networkVoteMessage{
			from: data.peerID,
			msg:  data.message,
		}
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
	for _, authorityIDToData := range vt.mapping {
		for _, data := range authorityIDToData {
			message := networkVoteMessage{
				from: data.peerID,
				msg:  data.message,
			}
			messages = append(messages, message)
		}
	}
	return messages
}
