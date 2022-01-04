// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// TransactionSource represents source of Transaction
type TransactionSource uint8

const (
	// TxnInBlock indicates transaction is already included in block.
	//
	// This means that we can't really tell where the transaction is coming from,
	// since it's already in the received block. Note that the custom validation logic
	// using either `Local` or `External` should most likely just allow `InBlock`
	// transactions as well.
	TxnInBlock TransactionSource = iota

	// TxnLocal indicates transaction is coming from a local source.
	//
	// This means that the transaction was produced internally by the node
	// (for instance an Off-Chain Worker, or an Off-Chain Call), as opposed
	// to being received over the network.
	TxnLocal

	// TxnExternal indicates transaction has been received externally.
	//
	// This means the transaction has been received from (usually) "untrusted" source,
	// for instance received over the network or RPC.
	TxnExternal
)

// TransactionPaymentQueryInfo represents the basic information of a given encoded extrinsic
type TransactionPaymentQueryInfo struct {
	Weight uint64
	// Class could be Normal (0), Operational (1), Mandatory (2)
	Class      int
	PartialFee *scale.Uint128
}
