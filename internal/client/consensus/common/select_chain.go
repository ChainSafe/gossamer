package common

import "github.com/ChainSafe/gossamer/internal/primitives/runtime"

// / The SelectChain trait defines the strategy upon which the head is chosen
// / if multiple forks are present for an opaque definition of "best" in the
// / specific chain build.
// /
// / The Strategy can be customized for the two use cases of authoring new blocks
// / upon the best chain or which fork to finalize.
type SelectChain[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]] interface {
	// Get all leaves of the chain, i.e. block hashes that have no children currently.
	// Leaves that can never be finalized will not be returned.
	Leaves() <-chan struct {
		Leaves []H
		Error  error
	}

	// Among those leaves deterministically pick one chain as the generally
	// best chain to author new blocks upon and probably (but not necessarily)
	// finalize.
	BestChain() <-chan HeaderError[H, N, Header]

	// Get the best descendent of baseHash that we should attempt to
	// finalize next, if any. It is valid to return the given baseHash
	// itself if no better descendent exists.
	FinalityTarget(baseHash H, maybeMaxNumber *N) <-chan HashError[H]
}

type HeaderError[H runtime.Hash, N runtime.Number, Header runtime.Header[N, H]] struct {
	Header Header
	Error  error
}

type HashError[H runtime.Hash] struct {
	Hash  H
	Error error
}
