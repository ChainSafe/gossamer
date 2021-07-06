package types

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// AccountInfo Information of an account.
type AccountInfo struct {
	// The number of transactions this account has sent.
	Nonce     uint32
	Consumers uint32
	Producers uint32
	// The additional data that belongs to this account. Used to store the balance(s) in a lot of chains.
	Data struct {
		Free       *scale.Uint128
		Reserved   *scale.Uint128
		MiscFrozen *scale.Uint128
		FreeFrozen *scale.Uint128
	}
}
