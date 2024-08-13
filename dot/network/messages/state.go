// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package messages

import (
	"fmt"

	pb "github.com/ChainSafe/gossamer/dot/network/proto"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"google.golang.org/protobuf/proto"
)

var _ P2PMessage = (*StateRequest)(nil)

// StateRequest defines the parameters to request the state keys
// and values from another peer
type StateRequest struct {
	Block   common.Hash
	Start   [][]byte
	NoProof bool
}

func (s *StateRequest) String() string {
	return fmt.Sprintf("StateRequest Block=%s Start=[0x%x, 0x%x] NoProof=%v",
		s.Block.String(),
		s.Start[0], s.Start[1],
		s.NoProof,
	)
}

func (s *StateRequest) Encode() ([]byte, error) {
	message := &pb.StateRequest{
		Block:   s.Block.ToBytes(),
		Start:   s.Start[:],
		NoProof: s.NoProof,
	}

	return proto.Marshal(message)
}

func (s *StateRequest) Decode(in []byte) error {
	message := &pb.StateRequest{}
	err := proto.Unmarshal(in, message)
	if err != nil {
		return err
	}

	s.Block = common.BytesToHash(message.Block)
	s.Start = make([][]byte, len(message.Start))
	copy(s.Start, message.Start)
	s.NoProof = message.NoProof
	return nil
}

type StateResponse struct {
	Entries []KeyValueStateEntry
	Proof   []byte
}

type KeyValueStateEntry struct {
	StateRoot    common.Hash
	StateEntries trie.Entries
	Complete     bool
}

func (s *StateResponse) Decode(in []byte) error {
	decodedResponse := &pb.StateResponse{}
	err := proto.Unmarshal(in, decodedResponse)
	if err != nil {
		return err
	}

	s.Proof = make([]byte, len(decodedResponse.Proof))
	copy(s.Proof, decodedResponse.Proof)

	s.Entries = make([]KeyValueStateEntry, len(decodedResponse.Entries))
	for idx, entry := range decodedResponse.Entries {
		s.Entries[idx] = KeyValueStateEntry{
			Complete:  entry.Complete,
			StateRoot: common.BytesToHash(entry.StateRoot),
		}

		trieFragment := make(trie.Entries, len(entry.Entries))
		for stateEntryIdx, stateEntry := range entry.Entries {
			trieFragment[stateEntryIdx] = trie.Entry{
				Key:   make([]byte, len(stateEntry.Key)),
				Value: make([]byte, len(stateEntry.Value)),
			}

			copy(trieFragment[stateEntryIdx].Key, stateEntry.Key)
			copy(trieFragment[stateEntryIdx].Value, stateEntry.Value)
		}

		s.Entries[idx].StateEntries = trieFragment
	}

	return nil
}
