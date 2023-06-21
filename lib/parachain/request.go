package parachain

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Request for fetching a large statement via request/response.
type StatementFetchingRequest struct {
	// Data needed to locate and identify the needed statement.
	RelayParent common.Hash `scale:"1"`

	// Hash of candidate that was used create the `CommitedCandidateRecept`.
	CandidateHash CandidateHash `scale:"2"`
}

// Encode returns the SCALE encoding of the StatementFetchingRequest.
func (s *StatementFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(*s)
}
