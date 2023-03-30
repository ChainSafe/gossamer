package network

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type WarpSyncProofRequestMessage struct {
	Begin common.Hash
}

func (w *WarpSyncProofRequestMessage) String() string {
	return fmt.Sprintf("WarpSyncProofRequestMessage Begin=%v", w.Begin)
}

func (w *WarpSyncProofRequestMessage) Encode() ([]byte, error) {
	return scale.Marshal(*w)
}

func (w *WarpSyncProofRequestMessage) Decode(in []byte) error {
	panic("not implemented yet")
}

type Vote struct {
	Hash   common.Hash
	Number uint32
}

type SignedVote struct {
	Vote        Vote
	Signature   [64]byte
	AuthorityID ed25519.PublicKeyBytes
}

// Commit contains all the signed precommits for a given block
type Commit struct {
	Hash       common.Hash
	Number     uint32
	Precommits []SignedVote
}

// Justification represents a finality justification for a block
type Justification struct {
	Round  uint64
	Commit Commit
}

type WarpSyncFragment struct {
	Header        types.Header
	Justification Justification
}

type WarpSyncProofResponse struct {
	Fragments  []WarpSyncFragment
	IsFinished bool
}

func (w *WarpSyncProofResponse) Encode() ([]byte, error) { return nil, nil }
func (w *WarpSyncProofResponse) Decode(in []byte) error {
	return scale.Unmarshal(in, w)
}
