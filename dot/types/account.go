package types

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// AccountInfo Information of an account.
type AccountInfo struct {
	// The number of transactions this account has sent.
	Nonce     uint32
	Consumers uint32
	Producers uint32
	// The additional data that belongs to this account. Used to store the balance(s) in a lot of chains.
	Data struct {
		Free       common.Uint128
		Reserved   common.Uint128
		MiscFrozen common.Uint128
		FreeFrozen common.Uint128
	}
}
