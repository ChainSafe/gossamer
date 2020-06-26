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

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/scale"
)

const specVersion uint32 = 193      // encoded as additional singed data when building UncheckedExtrinsic
const transactionVersion uint32 = 1 // encoded as additional singed data when building UncheckedExtrinsic

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

// pallet_system calls
const (
	SYS_fill_block Pallet = iota
	SYS_remark
	SYS_set_heap_pages
	SYS_set_code
	SYS_set_code_without_checks
	SYS_set_changes_trie_config
	SYS_set_storage
	SYS_kill_storage
	SYS_kill_prefix
	SYS_suicide
)

// Function struct to represent extrinsic call function
type Function struct {
	Call     Call
	Pallet   Pallet
	CallData interface{}
}
type Signature struct {
	Address common.Address
	Sig []byte
	Extra []byte

}
// UncheckedExtrinsic generic implementation of pre-verification extrinsic
type UncheckedExtrinsic struct {
	Signature Signature
	Function  Function
}

// CreateUncheckedExtrinsic builds UncheckedExtrinsic given function interface, index, genesisHash and Keypair
func CreateUncheckedExtrinsic(fnct interface{}, index *big.Int, genesisHash common.Hash, signer crypto.Keypair) (*UncheckedExtrinsic, error) {
	fnc, err := buildFunction(fnct)
	if err != nil {
		return nil, err
	}
	extra := struct {
		Nonce                    *big.Int
		ChargeTransactionPayment *big.Int
	}{
		index,
		big.NewInt(0),
	}
	additional := struct {
		SpecVersion        uint32
		TransactionVersion uint32
		GenesisHash        common.Hash
		GenesisHash2       common.Hash
	}{specVersion, transactionVersion, genesisHash, genesisHash}

	rawPayload := fromRaw(fnc, extra, additional)
	rawEnc, err := rawPayload.Encode()
	if err != nil {
		return nil, err
	}

	sig, err := signer.Sign(rawEnc)
	if err != nil {
		return nil, err
	}

	extraEnc, err := scale.Encode(extra)
	if err != nil {
		return nil, err
	}
	extraEnc = append([]byte{0}, extraEnc...) // todo determine what this represents

	signature := Signature{
		Address: signer.Public().Address(),
		Sig: sig,
		Extra: extraEnc,
	}
	ux := &UncheckedExtrinsic{
		Function:  *fnc,
		Signature: signature,
	}
	return ux, nil
}

// CreateUncheckedExtrinsicUnsigned to build unsigned extrinsic
func CreateUncheckedExtrinsicUnsigned(fnct interface{}) (*UncheckedExtrinsic, error) {
	fnc, err := buildFunction(fnct)
	if err != nil {
		return nil, err
	}

	ux := &UncheckedExtrinsic{
		Function: *fnc,
	}
	return ux, nil
}

func buildFunction(fnct interface{}) (*Function, error) {
	switch v := fnct.(type) {
	case *Transfer:
		return &Function{
			Call:     Balances,
			Pallet:   PB_Transfer,
			CallData: fnct,
		}, nil
	case *StorageChangeExt:
		return &Function{
			Call:     System,
			Pallet:   SYS_set_storage,
			CallData: fnct,
		}, nil
	default:
		return nil, fmt.Errorf("could not build Function for type %T", v)
	}
}

// Encode scale encode UncheckedExtrinsic
func (ux *UncheckedExtrinsic) Encode() ([]byte, error) {
	enc := []byte{}
	//enc = append(enc, []byte{45, 2, 132, 255}...) // TODO determine what this represents
	//enc = append(enc, ux.Signed...)
	//enc = append(enc, []byte{1}...) // TODO determine what this represents
	//enc = append(enc, ux.Signature...)
	//enc = append(enc, ux.Extra...)
	sigEnc, err := ux.Signature.Encode()
	if err != nil {
		return nil, err
	}
	enc = append(enc, sigEnc...)

	fncEnc, err := ux.Function.Encode()
	if err != nil {
		return nil, err
	}
	enc = append(enc, fncEnc...)
	return enc, nil
}

func (s *Signature) Encode() ([]byte, error) {
	return scale.Encode(s)
}
// Encode scale encode the UncheckedExtrinsic
func (f *Function) Encode() ([]byte, error) {
	switch f.Call {
	case Balances:
		// encode Balances type call
		return f.encodeBalance()
	case System:
		return f.encodeSystem()
	}
	return nil, nil
}

func (f *Function) encodeSystem() ([]byte, error) {
	enc := []byte{}
	enc = append(enc, byte(f.Call))
	switch f.Pallet {
	case SYS_set_storage:
		enc = append(enc, byte(f.Pallet))
		//enc = append(enc, byte(255)) // TODO not sure why this is used, research
		t := f.CallData.(*StorageChangeExt)
		kEnc, err := scale.Encode(t.key)
		if err != nil {
			return nil, err
		}
		fmt.Printf("kEnc %v\n", kEnc)
		enc = append(enc, kEnc...)
		if t.value.Exists() {
			vEnc, err := scale.Encode(t.value.Value())
			if err != nil {
				return nil, err
			}
			fmt.Printf("vEnc %v\n", vEnc)
			enc = append(enc, vEnc...)
		}
	}
	return enc, nil
}
func (f *Function) encodeBalance() ([]byte, error) {
	enc := []byte{}
	enc = append(enc, byte(f.Call))
	switch f.Pallet {
	case PB_Transfer:
		enc = append(enc, byte(f.Pallet))
		enc = append(enc, byte(255)) // TODO not sure why this is used, research
		t := f.CallData.(*Transfer)
		enc = append(enc, t.to[:]...)

		amtEnc, err := scale.Encode(big.NewInt(int64(t.amount)))
		if err != nil {
			return nil, err
		}
		enc = append(enc, amtEnc...)

	default:
		return nil, fmt.Errorf("could not encode pallet %v", f.Pallet)
	}
	return enc, nil
}

type signedPayload struct {
	Function       Function
	Extra          interface{}
	AdditionSigned interface{}
}

func fromRaw(fnc *Function, extra interface{}, additional interface{}) signedPayload {
	return signedPayload{
		Function:       *fnc,
		Extra:          extra,
		AdditionSigned: additional,
	}
}

// Encode scale encode SignedPayload
func (sp *signedPayload) Encode() ([]byte, error) {
	enc, err := sp.Function.Encode()
	if err != nil {
		return nil, err
	}
	//enc = append(enc, []byte{0}...) // TODO, determine why this byte is added

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
