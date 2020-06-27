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

// TODO determine how to get these values from the Runtime
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

// session calls
const (
	SESS_set_keys Pallet = iota
)

// Function struct to represent extrinsic call function
type Function struct {
	Call     Call
	Pallet   Pallet
	CallData interface{}
}

// Signature struct to represent signature parts
type Signature struct {
	Address []byte
	Sig     []byte
	Extra   []byte
}

// UncheckedExtrinsic generic implementation of pre-verification extrinsic
type UncheckedExtrinsic struct {
	Signature Signature
	Function  Function
}

// CreateUncheckedExtrinsic builds UncheckedExtrinsic given function interface, index, genesisHash and Keypair
func CreateUncheckedExtrinsic(fnc *Function, index *big.Int, genesisHash common.Hash, signer crypto.Keypair) (*UncheckedExtrinsic, error) {
	//fnc, err := buildFunction(fnct)
	//if err != nil {
	//	return nil, err
	//}
	extra := struct {
		Era                      [1]byte // TODO determine how Era is determined (Immortal is [1]byte{0}, Mortal is [2]byte{X, 0}, Need to determine how X is calculated)
		Nonce                    *big.Int
		ChargeTransactionPayment *big.Int
	}{
		[1]byte{0},
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
	fmt.Printf("encode extra %v\n", extraEnc)
	// TODO this changes mortality, determine how to set
	//extraEnc = append([]byte{22}, extraEnc...)

	signature := Signature{
		Address: signer.Public().Encode(),
		Sig:     sig,
		Extra:   extraEnc,
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
	case *AuthoritiesChangeExt:
		return &Function{
			Call:     Session,
			Pallet:   SESS_set_keys,
			CallData: fnct,
		}, nil
	default:
		return nil, fmt.Errorf("could not build Function for type %T", v)
	}
}

// Encode scale encode UncheckedExtrinsic
func (ux *UncheckedExtrinsic) Encode() ([]byte, error) {
	enc := []byte{}
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

	sEnc, err := scale.Encode(enc)
	if err != nil {
		return nil, err
	}

	return sEnc, nil
}

// Encode to encode Signature type
func (s *Signature) Encode() ([]byte, error) {
	enc := []byte{}
	//TODO determine why this 255 byte is here
	addEnc, err := scale.Encode(append([]byte{255}, s.Address...))
	if err != nil {
		return nil, err
	}
	enc = append(enc, addEnc...)
	// TODO find better way to handle keytype
	enc = append(enc, []byte{1}...) //this seems to represent signing key type 0 - Ed25519, 1 - Sr22219, 2 - Ecdsa
	enc = append(enc, s.Sig...)
	enc = append(enc, s.Extra...)
	return enc, nil
}

// Encode scale encode the UncheckedExtrinsic
func (f *Function) Encode() ([]byte, error) {
	switch f.Call {
	case Balances:
		// encode Balances type call
		enc := []byte{byte(f.Call), byte(f.Pallet)}
		dataEnc, err := scale.Encode(f.CallData)
		if err != nil {
			return nil, err
		}
		return append(enc, dataEnc...), nil
	case System:
		return f.encodeSystem()
	case Session:
		return f.encodeSession()
	}
	return nil, nil
}

func (f *Function) encodeSystem() ([]byte, error) {
	enc := []byte{}
	enc = append(enc, byte(f.Call))
	switch f.Pallet {
	case SYS_set_storage:
		enc = append(enc, byte(f.Pallet))
		t := f.CallData.(*StorageChangeExt)
		kEnc, err := scale.Encode(t.key)
		if err != nil {
			return nil, err
		}
		enc = append(enc, kEnc...)
		if t.value.Exists() {
			vEnc, err := scale.Encode(t.value.Value())
			if err != nil {
				return nil, err
			}
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

func (f *Function) encodeSession() ([]byte, error) {
	enc := []byte{}
	enc = append(enc, byte(f.Call))
	switch f.Pallet {
	case SESS_set_keys:
		enc = append(enc, byte(f.Pallet))
		t := f.CallData.(*AuthoritiesChangeExt)
		fmt.Printf("Session call data %v\n", t)
		acEnc, err := t.Encode()
		if err != nil {
			return nil, err
		}
		fmt.Printf("AC enc %v\n", acEnc)
		tstEnc, err := scale.Encode(t.authorityIDs[0])
		fmt.Printf("TestEnc %v\n", tstEnc)
		enc = append(enc, tstEnc...)
		//kEnc, err := scale.Encode(t.key)
		//if err != nil {
		//	return nil, err
		//}
		//fmt.Printf("kEnc %v\n", kEnc)
		//enc = append(enc, kEnc...)
		//if t.value.Exists() {
		//	vEnc, err := scale.Encode(t.value.Value())
		//	if err != nil {
		//		return nil, err
		//	}
		//	fmt.Printf("vEnc %v\n", vEnc)
		//	enc = append(enc, vEnc...)
		//}
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

	exEnc, err := scale.Encode(sp.Extra)
	if err != nil {
		return nil, err
	}

	// TODO this changes mortality, determine how to set
	//exEnc = append([]byte{22}, exEnc...)  // testing era
	enc = append(enc, exEnc...)

	addEnc, err := scale.Encode(sp.AdditionSigned)
	if err != nil {
		return nil, err
	}
	enc = append(enc, addEnc...)

	return enc, nil
}
