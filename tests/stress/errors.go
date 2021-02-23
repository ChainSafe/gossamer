package stress

import (
	"errors"
)

//nolint
var (
	errFinalizedBlockMismatch = errors.New("node finalized head hashes don't match")
	errNoFinalizedBlock       = errors.New("did not finalize block for round")
	errNoBlockAtNumber        = errors.New("no blocks found for given number")
	errBlocksAtNumberMismatch = errors.New("different blocks found for given number")
	errChainHeadMismatch      = errors.New("node chain head hashes don't match")
)
