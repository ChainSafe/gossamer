package types

import gtypes "github.com/centrifuge/go-substrate-rpc-client/v2/types"

// AccountInfo Information of an account.
type AccountInfo struct {
	// The number of transactions this account has sent.
	Nonce gtypes.U32
	// The number of other modules that currently depend on this account's existence. The account
	// cannot be reaped until this is zero.
	RefCount gtypes.U32
	// The additional data that belongs to this account. Used to store the balance(s) in a lot of chains.
	Data struct {
		Free       gtypes.U128
		Reserved   gtypes.U128
		MiscFrozen gtypes.U128
		FreeFrozen gtypes.U128
	}
}
