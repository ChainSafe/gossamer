package clean

import (
	"container/ring"

	"github.com/ChainSafe/gossamer/lib/common"
)

type CommitsCleaner struct {
	hashesRing *ring.Ring
	cleanup    func(hash common.Hash)
}

func NewCommitsCleaner(maxSize int,
	cleanup func(hash common.Hash)) *CommitsCleaner {
	return &CommitsCleaner{
		hashesRing: ring.New(maxSize),
		cleanup:    cleanup,
	}
}

func (cc *CommitsCleaner) TrackAndClean(hash common.Hash) {
	if cc.hashesRing.Value != nil {
		oldHash := cc.hashesRing.Value.(common.Hash)
		cc.cleanup(oldHash)
	}
	cc.hashesRing.Value = hash
	cc.hashesRing = cc.hashesRing.Next()
}
