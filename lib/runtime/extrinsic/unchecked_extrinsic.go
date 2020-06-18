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
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/stretchr/testify/require"
	"math/big"
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

type Function struct {
	Call  Call
	Pallet    Pallet
	CallData  interface{}
}
// UncheckedExtrinsic generic implementation of pre-verification extrinsic
type UncheckedExtrinsic struct {
	Signed []byte
	Signature []byte
	Extra []byte
	Function Function
}

func CreateUncheckedExtrinsic(fnct interface{}, index *big.Int, genesisHash common.Hash) UncheckedExtrinsic {
	fnc := buildFunction(fnct)
	ux := UncheckedExtrinsic{}

	extra := struct {
		Nonce *big.Int
		ChargeTransactionPayment *big.Int
	}{
		index,
		big.NewInt(0),
	}
	additional := struct {
		SpecVersion uint32
		TransacionVersion uint32
		GenesisHash common.Hash
		GenesisHash2 common.Hash
	}{252, 1, genesisHash, genesisHash}
	rawPayload := fromRaw(fnc, extra, additional)
	rawEnc, err := rawPayload.Encode()
	require.NoError(t, err)
	fmt.Printf("RAW ENC %v\n", rawEnc)

	return ux
}

func buildFunction(fnct interface{}) *Function {
	// TODO make this build the function
	return &Function{
		Call:     Balances,
		Pallet:   PB_Transfer,
		CallData: fnct,
	}
}

func (ux *UncheckedExtrinsic) Encode() ([]byte, error) {
	enc := []byte{}
	enc = append(enc, []byte{45, 2, 132, 255}...)
	enc = append(enc, ux.Signed...)
	enc = append(enc, []byte{1}...)   // TODO determine what this represents
	enc = append(enc, ux.Signature...)
	enc = append(enc, ux.Extra...)
	fncEnc, err := ux.Function.Encode()
	if err != nil {
		return nil, err
	}
	enc = append(enc, fncEnc...)
	return enc, nil
}

// Encode scale encode the UncheckedExtrinsic
func (f *Function) Encode() ([]byte, error) {
	switch f.Call {
	case Balances:
		// encode Balances type call
		return f.encodeBalance()
	}
	return nil, nil
}

func (f *Function) encodeBalance() ([]byte, error) {
	enc := []byte{}
	enc = append(enc, byte(f.Call))
	switch f.Pallet {
	case PB_Transfer:
		enc = append(enc, byte(f.Pallet))
		enc = append(enc, byte(255)) // TODO not sure why this is used, research
		t := f.CallData.(Transfer)
		enc = append(enc, t.to[:]...)

		amtEnc, err := scale.Encode(big.NewInt(int64(t.amount))) // TODO, research why amount needs bigInt encoding (not uint64)
		if err != nil {
			return nil, err
		}
		enc = append(enc, amtEnc...)
		//enc = append([]byte{byte(4)}, enc...) // TODO not sure why this needs a 4 here, research

		//enc, err = scale.Encode(enc)
		//if err != nil {
		//	return nil, err
		//}
	default:
		return nil, fmt.Errorf("could not encode pallet %v", f.Pallet)
	}
	return enc, nil
}


//func (ue *UncheckedExtrinsic)Sign(key *sr25519.PrivateKey) {
	//msg, err := ue.Encode()
	//fmt.Printf("tran enc %x\n", msg)
	//
	//sig, err := key.Sign(msg)
	//if err != nil {
	//	//return nil, err
	//}
	//
	//sigb := [64]byte{}
	//copy(sigb[:], sig)
	//fmt.Printf("Sigb %v\n", sigb)
	//ue.Signature = sigb
//}

type SignedPayload struct {
	Function Function
	Extra interface{}
	AdditionSigned interface{}
}
func fromRaw(fnc Function, extra interface{}, additional interface{}) SignedPayload {
	return SignedPayload{
		Function:  fnc,
		Extra: extra,
		AdditionSigned: additional,
	}
}

func (sp *SignedPayload) Encode() ([]byte, error) {
	enc, err := sp.Function.Encode()
	if err != nil {
		return nil, err
	}
	enc = append(enc, []byte{0}...)  // TODO, determine why this byte is added

	exEnc, err := scale.Encode(sp.Extra)
	if err != nil {
		return nil, err
	}
	enc = append(enc, exEnc...)

	addEnc, err := scale.Encode(sp.AdditionSigned)
	if err != nil {
		return nil, err
	}
	enc = append(enc, addEnc...)

	return enc, nil
}