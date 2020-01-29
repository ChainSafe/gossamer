package types

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/common/optional"
)

func TestBlockDataEncodeEmpty(t *testing.T) {
	hash := common.NewHash([]byte{0})

	bd := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(false, nil),
		Body:          optional.NewBody(false, nil),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	expected := append([]byte{0}, hash[:]...)
	expected = append(expected, []byte{0, 0, 0, 0}...)

	enc, err := bd.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expected, enc) {
		t.Fatalf("Fail: got %x expected %x", enc, expected)
	}
}

func TestBlockDataEncodeHeader(t *testing.T) {
	hash := common.NewHash([]byte{0})
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	header := &optional.CoreHeader{
		ParentHash:     testHash,
		Number:         big.NewInt(1),
		StateRoot:      testHash,
		ExtrinsicsRoot: testHash,
		Digest:         [][]byte{{0xe, 0xf}},
	}

	bd := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(true, header),
		Body:          optional.NewBody(false, nil),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	expected, err := common.HexToBytes("0x000000000000000000000000000000000000000000000000000000000000000001000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04080e0f00000000")
	if err != nil {
		t.Fatal(err)
	}

	enc, err := bd.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expected, enc) {
		t.Fatalf("Fail: got %x expected %x", enc, expected)
	}
}

func TestBlockDataEncodeBody(t *testing.T) {
	hash := common.NewHash([]byte{0})
	body := optional.CoreBody{0xa, 0xb, 0xc, 0xd}

	bd := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(false, nil),
		Body:          optional.NewBody(true, body),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	expected, err := common.HexToBytes("0x00000000000000000000000000000000000000000000000000000000000000000001100a0b0c0d000000")
	if err != nil {
		t.Fatal(err)
	}

	enc, err := bd.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expected, enc) {
		t.Fatalf("Fail: got %x expected %x", enc, expected)
	}
}

func TestBlockDataEncodeAll(t *testing.T) {
	hash := common.NewHash([]byte{0})
	body := optional.CoreBody{0xa, 0xb, 0xc, 0xd}

	bd := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(false, nil),
		Body:          optional.NewBody(true, body),
		Receipt:       optional.NewBytes(true, []byte("asdf")),
		MessageQueue:  optional.NewBytes(true, []byte("ghjkl")),
		Justification: optional.NewBytes(true, []byte("qwerty")),
	}

	expected, err := common.HexToBytes("0x00000000000000000000000000000000000000000000000000000000000000000001100a0b0c0d011061736466011467686a6b6c0118717765727479")
	if err != nil {
		t.Fatal(err)
	}

	enc, err := bd.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expected, enc) {
		t.Fatalf("Fail: got %x expected %x", enc, expected)
	}
}

func TestBlockDataDecodeHeader(t *testing.T) {
	hash := common.NewHash([]byte{0})
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})

	header := &optional.CoreHeader{
		ParentHash:     testHash,
		Number:         big.NewInt(1),
		StateRoot:      testHash,
		ExtrinsicsRoot: testHash,
		Digest:         [][]byte{{0xe, 0xf}},
	}

	expected := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(true, header),
		Body:          optional.NewBody(false, nil),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	enc, err := common.HexToBytes("0x000000000000000000000000000000000000000000000000000000000000000001000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04080e0f00000000")
	if err != nil {
		t.Fatal(err)
	}

	res := new(BlockData)
	r := &bytes.Buffer{}
	r.Write(enc)

	err = res.Decode(r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, expected) {
		t.Fatalf("Fail: got %v expected %v", res, expected)
	}
}

func TestBlockDataDecodeBody(t *testing.T) {
	hash := common.NewHash([]byte{0})
	body := optional.CoreBody{0xa, 0xb, 0xc, 0xd}

	expected := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(false, nil),
		Body:          optional.NewBody(true, body),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	enc, err := common.HexToBytes("0x00000000000000000000000000000000000000000000000000000000000000000001100a0b0c0d000000")
	if err != nil {
		t.Fatal(err)
	}

	res := new(BlockData)
	r := &bytes.Buffer{}
	r.Write(enc)

	err = res.Decode(r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, expected) {
		t.Fatalf("Fail: got %v expected %v", res, expected)
	}
}

func TestBlockDataDecodeAll(t *testing.T) {
	hash := common.NewHash([]byte{0})
	body := optional.CoreBody{0xa, 0xb, 0xc, 0xd}

	expected := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(false, nil),
		Body:          optional.NewBody(true, body),
		Receipt:       optional.NewBytes(true, []byte("asdf")),
		MessageQueue:  optional.NewBytes(true, []byte("ghjkl")),
		Justification: optional.NewBytes(true, []byte("qwerty")),
	}

	enc, err := common.HexToBytes("0x00000000000000000000000000000000000000000000000000000000000000000001100a0b0c0d011061736466011467686a6b6c0118717765727479")
	if err != nil {
		t.Fatal(err)
	}

	res := new(BlockData)
	r := &bytes.Buffer{}
	r.Write(enc)

	err = res.Decode(r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(res, expected) {
		t.Fatalf("Fail: got %v expected %v", res, expected)
	}
}
