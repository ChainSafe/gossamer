// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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
