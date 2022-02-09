// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
)

type pubKeyToSignedVote struct {
	mutex   sync.RWMutex
	mapping map[ed25519.PublicKeyBytes]*SignedVote
}

func newPubKeyToSignedVote() *pubKeyToSignedVote {
	return &pubKeyToSignedVote{
		mapping: make(map[ed25519.PublicKeyBytes]*SignedVote),
	}
}

func (p *pubKeyToSignedVote) clear() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.mapping = make(map[ed25519.PublicKeyBytes]*SignedVote)
}

func (p *pubKeyToSignedVote) get(publicKey ed25519.PublicKeyBytes) (
	signedVote *SignedVote) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.mapping[publicKey]
}

func (p *pubKeyToSignedVote) set(publicKey ed25519.PublicKeyBytes,
	signedVote *SignedVote) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.mapping[publicKey] = signedVote
}

func (p *pubKeyToSignedVote) delete(publicKey ed25519.PublicKeyBytes) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.mapping, publicKey)
}

func (p *pubKeyToSignedVote) len() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return len(p.mapping)
}

func (p *pubKeyToSignedVote) makeJustifications(bfc common.Hash,
	blockState BlockState) (justifications []SignedVote, err error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, signedVote := range p.mapping {
		isDescendant, err := blockState.IsDescendantOf(bfc, signedVote.Vote.Hash)
		if err != nil {
			return nil, fmt.Errorf("cannot verify descendance of %s for parent bfc %s: %w",
				signedVote.Vote.Hash, bfc, err)
		}

		if isDescendant {
			justifications = append(justifications, *signedVote)
		}
	}

	return justifications, nil
}

func (p *pubKeyToSignedVote) getDirectVotes() (votes map[Vote]uint64) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	votes = make(map[Vote]uint64, len(p.mapping))
	for _, signedVote := range p.mapping {
		votes[signedVote.Vote]++
	}

	return votes
}

func (p *pubKeyToSignedVote) getPreVotes() (preVotes []ed25519.PublicKeyBytes) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	preVotes = make([]ed25519.PublicKeyBytes, 0, len(p.mapping))
	for publicKey := range p.mapping {
		preVotes = append(preVotes, publicKey)
	}

	return preVotes
}
