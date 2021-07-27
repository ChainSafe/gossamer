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
	"bytes"
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"math/big"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"

	"github.com/stretchr/testify/require"
)

var testDigest = &Digest{
	&PreRuntimeDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{1, 2, 3},
	},
	&SealDigest{
		ConsensusEngineID: BabeEngineID,
		Data:              []byte{4, 5, 6, 7},
	},
}

func TestNilBlockData(t *testing.T) {
	expected, err := common.HexToBytes("0x00000000000000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	bd := BlockDataVdt{
		Hash:          common.NewHash([]byte{0}),
		Header:        nil,
		Body:          nil,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}

	enc, err := scale.Marshal(bd)
	require.NoError(t, err)

	require.Equal(t, expected, enc)
}

func TestFullBlockDataNew(t *testing.T) {
	expected, err := common.HexToBytes("0x7d0000000000000000000000000000000000000000000000000000000000000001000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f04000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f000102030405060708090a0b0c0d0e0f0806424142450c0102030542414245100405060701100a0b0c0d010401010402010403")
	require.NoError(t, err)

	hash := common.NewHash([]byte{125})
	testHash := common.NewHash([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf})
	body := NewBody([]byte{0xa, 0xb, 0xc, 0xd})

	vdts := DigestVdtSlice
	err = vdts.Add(
		PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              []byte{1, 2, 3},
		},
		SealDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              []byte{4, 5, 6, 7},
		},
	)
	require.NoError(t, err)

	headerVdt, err := NewHeaderVdt(testHash, testHash, testHash, big.NewInt(1), vdts)
	require.NoError(t, err)

	bd := BlockDataVdt{
		Hash:          hash,
		Header:        headerVdt,
		Body:          body,
		Receipt:       &[]byte{1},
		MessageQueue:  &[]byte{2},
		Justification: &[]byte{3},
	}

	enc, err := scale.Marshal(bd)
	require.NoError(t, err)

	require.Equal(t, expected, enc)
}

func TestFullBlockData(t *testing.T) {
	hash := common.NewHash([]byte{0})
	bd := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(false, nil),
		Body:          optional.NewBody(false, nil),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}
	enc, err := bd.Encode()
	require.NoError(t, err)

	fmt.Println(common.BytesToHex(enc))
}

func TestSingleNonNilBlockData(t *testing.T) {
	vdts := DigestVdtSlice
	err := vdts.Add(
		PreRuntimeDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0201000000ef55a50f00000000"),
		},
		ConsensusDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x0118ca239392960473fe1bc65f94ee27d890a49c1b200c006ff5dcc525330ecc16770100000000000000b46f01874ce7abbb5220e8fd89bede0adad14c73039d91e28e881823433e723f0100000000000000d684d9176d6eb69887540c9a89fa6097adea82fc4b0ff26d1062b488f352e179010000000000000068195a71bdde49117a616424bdc60a1733e96acb1da5aeab5d268cf2a572e94101000000000000001a0575ef4ae24bdfd31f4cb5bd61239ae67c12d4e64ae51ac756044aa6ad8200010000000000000018168f2aad0081a25728961ee00627cfe35e39833c805016632bf7c14da5800901000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		},
		SealDigest{
			ConsensusEngineID: BabeEngineID,
			Data:              common.MustHexToBytes("0x4625284883e564bc1e4063f5ea2b49846cdddaa3761d04f543b698c1c3ee935c40d25b869247c36c6b8a8cbbd7bb2768f560ab7c276df3c62df357a7e3b1ec8d"),
		},
	)
	require.NoError(t, err)

	headerVdt, err := NewHeaderVdt(common.Hash{1}, common.Hash{1}, common.Hash{1}, big.NewInt(1), vdts)
	require.NoError(t, err)

	bd := BlockDataVdt{
		Hash:          common.NewHash([]byte{0}),
		Header:        headerVdt,
		Body:          NewBody([]byte{0xa, 0xb, 0xc, 0xd}),
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}

	enc, err := scale.Marshal(bd)
	require.NoError(t, err)

	block := BlockDataVdt{Header: NewEmptyHeaderVdt()}
	err = scale.Unmarshal(enc, &block)
	require.NoError(t, err)
	_ = block.Header.Hash()
	require.Equal(t, bd, block)
}

func TestVdtEncode(t *testing.T) {
	hash := common.NewHash([]byte{0})
	body := optional.CoreBody{0xa, 0xb, 0xc, 0xd}
	//body2 := NewBody([]byte{0xa, 0xb, 0xc, 0xd})

	bd := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(false, nil),
		Body:          optional.NewBody(true, body),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	bdVdt := BlockDataVdt{
		Hash:          common.NewHash([]byte{0}),
		Header:        nil,
		Body:          NewBody([]byte{0xa, 0xb, 0xc, 0xd}),
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}

	expected, err := common.HexToBytes("0x00000000000000000000000000000000000000000000000000000000000000000001100a0b0c0d000000")
	require.NoError(t, err)

	enc, err := bd.Encode()
	require.NoError(t, err)
	require.Equal(t, expected, enc)

	enc2, err := scale.Marshal(bdVdt)
	require.NoError(t, err)
	require.Equal(t, enc, enc2)

	// Decode
	var block BlockDataVdt
	err = scale.Unmarshal(enc2, &block)
	require.NoError(t, err)
	require.Equal(t, bdVdt, block)

	r := &bytes.Buffer{}
	_, _ = r.Write(enc)

	res := new(BlockData)
	err = res.Decode(r)
	require.NoError(t, err)
	require.Equal(t, bd, res)
}

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
		Digest:         testDigest,
	}

	bd := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(true, header),
		Body:          optional.NewBody(false, nil),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	enc, err := bd.Encode()
	require.NoError(t, err)

	r := &bytes.Buffer{}
	_, _ = r.Write(enc)

	res := new(BlockData)
	err = res.Decode(r)
	require.NoError(t, err)
	require.Equal(t, bd, res)
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
		Digest:         testDigest,
	}

	expected := &BlockData{
		Hash:          hash,
		Header:        optional.NewHeader(true, header),
		Body:          optional.NewBody(false, nil),
		Receipt:       optional.NewBytes(false, nil),
		MessageQueue:  optional.NewBytes(false, nil),
		Justification: optional.NewBytes(false, nil),
	}

	enc, err := expected.Encode()
	require.NoError(t, err)

	res := new(BlockData)
	r := &bytes.Buffer{}
	r.Write(enc)

	err = res.Decode(r)
	require.NoError(t, err)

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