package validatorside

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	CollatorV1 = 1
	CollatroV2 = 2
)

type prospectiveCandidate struct {
	CandidateHash      parachaintypes.CandidateHash
	ParentHeadDataHash common.Hash
}

type pendingCollation struct {
	RelayParent          common.Hash
	ParaID               parachaintypes.ParaID
	PeerID               peer.ID
	CommitmentHash       *common.Hash
	ProspectiveCandidate *prospectiveCandidate
}

type collationFetchRequest struct {
	ctx                 context.Context
	pendingCollation    pendingCollation
	collatorId          parachaintypes.CollatorID
	collatorProtVersion int
	response            CollationFetchingResponse
}

type collationFetchTimeout struct {
	collatorID      parachaintypes.CollatorID
	candidateHash   *parachaintypes.CandidateHash
	relayParentHash common.Hash
}
