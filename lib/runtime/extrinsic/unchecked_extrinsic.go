// Copyright 2020 ChainSafe Systems (ON) Corp.
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
package extrinsic

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/scale"
)

// Call interface for method extrinsic is calling
type Call byte

// consts for node_runtime calls
const (
	System Call = iota
	Utility
	Babe
	Timestamp
	Authorship
	Indices
	Balances
	Staking
	Session
)

// Pallet for runtime sub-calls
type Pallet int

// pallet_balances calls
const (
	PB_Transfer Pallet = iota
	PB_Set_balance
	PB_Force_transfer
	PB_Transfer_keep_alive
)

// UncheckedExtrinsic generic implementation of pre-verification extrinsic
type UncheckedExtrinsic struct {
	Signature interface{} // optional type Address, Signature, Extra
	Function  Call
	Pallet    Pallet
	CallData  interface{}
}

// Encode scale encode the UncheckedExtrinsic
func (ue *UncheckedExtrinsic) Encode() ([]byte, error) {
	switch ue.Function {
	case Balances:
		// encode Balances type call
		return ue.encodeBalance()
	}
	return nil, nil
}

func (ue *UncheckedExtrinsic) encodeBalance() ([]byte, error) {
	enc := []byte{}
	enc = append(enc, byte(ue.Function))
	switch ue.Pallet {
	case PB_Transfer:
		enc = append(enc, byte(ue.Pallet))
		enc = append(enc, byte(255)) // TODO not sure why this is used, research
		t := ue.CallData.(Transfer)
		enc = append(enc, t.to[:]...)

		amtEnc, err := scale.Encode(big.NewInt(int64(t.amount))) // TODO, research why amount needs bigInt encoding (not uint64)
		if err != nil {
			return nil, err
		}
		enc = append(enc, amtEnc...)
		enc = append([]byte{byte(4)}, enc...) // TODO not sure why this needs a 4 here, research

		enc, err = scale.Encode(enc)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("could not encode pallet %v", ue.Pallet)
	}
	return enc, nil
}
