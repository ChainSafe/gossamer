// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package crypto

import (
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"
)

// SigVerifyFunc verify an signature given a pubkey and msg
type SigVerifyFunc func(pubkey, sig, msg []byte) (bool, error)

// Signature ...
type Signature struct {
	PubKey     []byte
	Sign       []byte
	Msg        []byte
	VerifyFunc SigVerifyFunc
}

// SignatureVerifier ...
type SignatureVerifier struct {
	batch   []*Signature
	init    bool // Indicates whether the batch processing is started.
	invalid bool // Set to true if any signature verification fails.
	logger  log.LeveledLogger
	closeCh chan struct{}
	sync.RWMutex
	sync.Once
	sync.WaitGroup
}

// NewSignatureVerifier initialises SignatureVerifier which does background verification of signatures.
// Start() is called to start the verification process.
// Finish() is called to stop the verification process.
// Signatures can be added to the batch using Add().
func NewSignatureVerifier(logger log.LeveledLogger) *SignatureVerifier {
	return &SignatureVerifier{
		batch:   make([]*Signature, 0),
		init:    false,
		invalid: false,
		logger:  logger,
		RWMutex: sync.RWMutex{},
		closeCh: make(chan struct{}),
	}
}

// Start signature verification in batch.
func (sv *SignatureVerifier) Start() {
	// Update the init state.
	sv.Lock()
	sv.init = true
	sv.Unlock()

	sv.WaitGroup.Add(1)

	go func() {
		defer sv.Done()
		for {
			select {
			case <-sv.closeCh:
				return
			default:
				sig := sv.Remove()
				if sig == nil {
					continue
				}
				ok, err := sig.VerifyFunc(sig.PubKey, sig.Sign, sig.Msg)
				if err != nil || !ok {
					sv.logger.Errorf("[ext_crypto_start_batch_verify_version_1]: %s", err)
					sv.Invalid()
					return
				}
			}
		}
	}()
}

// IsStarted ...
func (sv *SignatureVerifier) IsStarted() bool {
	sv.RLock()
	defer sv.RUnlock()
	return sv.init
}

// IsInvalid ...
func (sv *SignatureVerifier) IsInvalid() bool {
	sv.RLock()
	defer sv.RUnlock()
	return sv.invalid
}

// Invalid ...
func (sv *SignatureVerifier) Invalid() {
	sv.RLock()
	defer sv.RUnlock()
	sv.invalid = true
}

// Add ...
func (sv *SignatureVerifier) Add(s *Signature) {
	if sv.IsInvalid() {
		return
	}

	sv.Lock()
	defer sv.Unlock()
	sv.batch = append(sv.batch, s)
}

// Remove returns the first signature from the batch. Returns nil if batch is empty.
func (sv *SignatureVerifier) Remove() *Signature {
	sv.Lock()
	defer sv.Unlock()
	if len(sv.batch) == 0 {
		return nil
	}
	sign := sv.batch[0]
	sv.batch = sv.batch[1:len(sv.batch)]
	return sign
}

// Reset reset the signature verifier for reuse.
func (sv *SignatureVerifier) Reset() {
	sv.Lock()
	defer sv.Unlock()
	sv.init = false
	sv.batch = make([]*Signature, 0)
	sv.invalid = false
	sv.closeCh = make(chan struct{})
}

// Finish waits till batch is finished. Returns true if all the signatures are valid, Otherwise returns false.
func (sv *SignatureVerifier) Finish() bool {
	for {
		time.Sleep(100 * time.Millisecond)
		sv.Lock()
		if sv.invalid || len(sv.batch) == 0 {
			close(sv.closeCh)
			sv.Unlock()
			break
		}
		sv.RUnlock()
	}
	// Wait till start function to finish and then reset it.
	sv.Wait()
	isInvalid := sv.IsInvalid()
	sv.Reset()
	return !isInvalid
}
