package extrinsic

import (
	"bytes"
	"reflect"
	"testing"
)

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
