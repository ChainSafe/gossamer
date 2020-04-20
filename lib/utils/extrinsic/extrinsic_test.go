package extrinsic

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

func TestAuthoritiesChangeExt_Encode(t *testing.T) {
	t.Skip()

	// TODO: scale isn't working for arrays of [32]byte
	kp, err := sr25519.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	pub := kp.Public().Encode()
	pubk := [32]byte{}
	copy(pubk[:], pub)

	authorities := [][32]byte{pubk}

	ext := NewAuthoritiesChangeExt(authorities)

	enc, err := ext.Encode()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(enc)

	r := &bytes.Buffer{}
	r.Write(enc)
	res, err := DecodeExtrinsic(r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, ext) {
		t.Fatalf("Fail: got %v expected %v", res, ext)
	}
}

func TestTransferExt_Encode(t *testing.T) {
	transfer := NewTransfer(8, 9, 1000, 1)
	sig := [64]byte{}
	ext := NewTransferExt(transfer, sig, false)

	enc, err := ext.Encode()
	if err != nil {
		t.Fatal(err)
	}

	r := &bytes.Buffer{}
	r.Write(enc)
	res, err := DecodeExtrinsic(r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, ext) {
		t.Fatalf("Fail: got %v expected %v", res, ext)
	}
}

func TestTransferExt_Decode(t *testing.T) {
	// from substrate test runtime
	enc := []byte{1, 212, 53, 147, 199, 21, 253, 211, 28, 97, 20, 26, 189, 4, 169, 159, 214, 130, 44, 133, 88, 133, 76, 205, 227, 154, 86, 132, 231, 165, 109, 162, 125, 144, 181, 171, 32, 92, 105, 116, 201, 234, 132, 27, 230, 136, 134, 70, 51, 220, 156, 168, 163, 87, 132, 62, 234, 207, 35, 20, 100, 153, 101, 254, 34, 69, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 188, 209, 252, 92, 63, 148, 77, 142, 7, 243, 22, 228, 44, 21, 71, 247, 32, 57, 110, 128, 88, 229, 128, 95, 235, 4, 133, 101, 81, 214, 227, 69, 107, 68, 73, 106, 89, 131, 124, 97, 135, 112, 48, 67, 75, 23, 59, 30, 79, 105, 27, 187, 25, 235, 9, 108, 125, 106, 199, 230, 199, 189, 76, 137, 0}
	r := &bytes.Buffer{}
	r.Write(enc)
	res, err := DecodeExtrinsic(r)
	if err != nil {
		t.Fatal(err)
	}

	// []byte{32, 57, 110, 128, 88, 229, 128, 95, 235, 4, 133, 101, 81, 214, 227, 69, 107, 68, 73, 106, 89, 131, 124, 97, 135, 112, 48, 67, 75, 23, 59, 30, 79, 105, 27, 187, 25, 235, 9, 108, 125, 106, 199, 230, 199, 189, 76, 137}
	// where does this go?

	from := binary.LittleEndian.Uint64([]byte{212, 53, 147, 199, 21, 253, 211, 28})
	to := binary.LittleEndian.Uint64([]byte{97, 20, 26, 189, 4, 169, 159, 214})
	amount := binary.LittleEndian.Uint64([]byte{130, 44, 133, 88, 133, 76, 205, 227})
	nonce := binary.LittleEndian.Uint64([]byte{154, 86, 132, 231, 165, 109, 162, 125, 144})

	expected := &TransferExt{
		transfer: &Transfer{
			from:   from,
			to:     to,
			amount: amount,
			nonce:  nonce,
		},
		signature:                    [sr25519.SignatureLength]byte{144, 181, 171, 32, 92, 105, 116, 201, 234, 132, 27, 230, 136, 134, 70, 51, 220, 156, 168, 163, 87, 132, 62, 234, 207, 35, 20, 100, 153, 101, 254, 34, 69, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 188, 209, 252, 92, 63, 148, 77, 142, 7, 243, 22, 228, 44, 21, 71, 247},
		exhaustResourcesWhenNotFirst: false,
	}

	var transfer *TransferExt
	var ok bool

	if transfer, ok = res.(*TransferExt); !ok {
		t.Fatal("Fail: got wrong extrinsic type")
	}

	if !reflect.DeepEqual(transfer, expected) {
		t.Fatalf("Fail: got %v expected %v", transfer.transfer, expected.transfer)
	}
}

func TestIncludeDataExt_Encode(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	ext := NewIncludeDataExt(data)

	enc, err := ext.Encode()
	if err != nil {
		t.Fatal(err)
	}

	r := &bytes.Buffer{}
	r.Write(enc)
	res, err := DecodeExtrinsic(r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, ext) {
		t.Fatalf("Fail: got %v expected %v", res, ext)
	}
}

func TestStorageChangeExt_Encode(t *testing.T) {
	key := []byte("noot")
	value := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	ext := NewStorageChangeExt(key, value)

	enc, err := ext.Encode()
	if err != nil {
		t.Fatal(err)
	}

	r := &bytes.Buffer{}
	r.Write(enc)
	res, err := DecodeExtrinsic(r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, ext) {
		t.Fatalf("Fail: got %v expected %v", res, ext)
	}
}
