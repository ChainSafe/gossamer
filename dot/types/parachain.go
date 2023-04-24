package types

import (
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

// ValidatorID represents a validator ID
type ValidatorID [sr25519.PublicKeyLength]byte

// Validator represents a validator
type Validator struct {
	Key crypto.PublicKey
}

// FromRawSr25519 sets the Validator given ValidatorID. It converts the byte representations of
// the authority public keys into a sr25519.PublicKey.
func (a *Validator) FromRawSr25519(id ValidatorID) error {
	key, err := sr25519.NewPublicKey(id[:])
	if err != nil {
		return err
	}

	a.Key = key
	return nil
}

// ValidatorIDToValidator turns a slice of ValidatorID into a slice of Validator
func ValidatorIDToValidator(ids []ValidatorID) ([]Validator, error) {
	validators := make([]Validator, len(ids))
	for i, r := range ids {
		validators[i] = Validator{}
		err := validators[i].FromRawSr25519(r)
		if err != nil {
			return nil, err
		}
	}

	return validators, nil
}

// ValidatorIndex represents a validator index
type ValidatorIndex uint32

// GroupRotationInfo represents the group rotation info
type GroupRotationInfo struct {
	// SessionStartBlock is the block number at which the session started
	SessionStartBlock uint64 `scale:"1"`
	// GroupRotationFrequency indicates how often groups rotate. 0 means never.
	GroupRotationFrequency uint64 `scale:"2"`
	// Now indicates the current block number.
	Now uint64 `scale:"3"`
}

type ValidatorGroups struct {
	// Validators is an array the validator set Ids
	Validators [][]ValidatorIndex `scale:"1"`
	// GroupRotationInfo is the group rotation info
	GroupRotationInfo GroupRotationInfo `scale:"2"`
}
