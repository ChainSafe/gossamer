package clean

import (
	"container/ring"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
)

type VotesCleaner struct {
	hashesRing *ring.Ring
	cleanup    func(hash common.Hash, authorityID ed25519.PublicKeyBytes)
}

func NewVotesCleaner(maxSize int,
	cleanup func(hash common.Hash, authorityID ed25519.PublicKeyBytes)) *VotesCleaner {
	return &VotesCleaner{
		hashesRing: ring.New(maxSize),
		cleanup:    cleanup,
	}
}

type hashAuthorityID struct {
	hash        common.Hash
	authorityID ed25519.PublicKeyBytes
}

func (vc *VotesCleaner) TrackAndClean(hash common.Hash, authorityID ed25519.PublicKeyBytes) {
	if vc.hashesRing.Value != nil {
		oldHashAuthorityID := vc.hashesRing.Value.(hashAuthorityID)
		oldHash := oldHashAuthorityID.hash
		oldAuthorityID := oldHashAuthorityID.authorityID
		vc.cleanup(oldHash, oldAuthorityID)
	}
	vc.hashesRing.Value = hashAuthorityID{
		hash:        hash,
		authorityID: authorityID,
	}
	vc.hashesRing = vc.hashesRing.Next()
}
