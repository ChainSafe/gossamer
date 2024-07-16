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
}

func (s *StateResponse) Decode(in []byte) error {
		
}

type KeyValueStateEntry struct {
	StateRoot    []byte
	StateEntries trie.Entries
	Complete     bool
}
