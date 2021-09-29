package types

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestEncodeGrandpaVote(t *testing.T) {
	exp := common.MustHexToBytes("0x0a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000")
	var testVote = GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}

	enc, err := scale.Marshal(testVote)
	require.NoError(t, err)
	require.Equal(t, exp, enc)

	dec := GrandpaVote{}
	err = scale.Unmarshal(enc, &dec)
	require.NoError(t, err)
	require.Equal(t, testVote, dec)

}
func TestEncodeSignedVote(t *testing.T) {
	exp := common.MustHexToBytes("0x0a0b0c0d00000000000000000000000000000000000000000000000000000000e7030000010203040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000506070800000000000000000000000000000000000000000000000000000000")
	var testVote = GrandpaVote{
		Hash:   common.Hash{0xa, 0xb, 0xc, 0xd},
		Number: 999,
	}
	var testSignature = [64]byte{1, 2, 3, 4}
	var testAuthorityID = [32]byte{5, 6, 7, 8}

	sv := GrandpaSignedVote{
		Vote:        testVote,
		Signature:   testSignature,
		AuthorityID: testAuthorityID,
	}
	enc, err := scale.Marshal(sv)
	require.NoError(t, err)
	require.Equal(t, exp, enc)

	res := GrandpaSignedVote{}
	err = scale.Unmarshal(enc, &res)
	require.NoError(t, err)
	require.Equal(t, sv, res)
}

func TestGrandpaAuthoritiesRaw(t *testing.T) {
	ad := new(GrandpaAuthoritiesRaw)
	buf := &bytes.Buffer{}
	data, _ := common.HexToBytes("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640000000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000")
	buf.Write(data)

	ad, err := ad.Decode(buf)
	require.NoError(t, err)
	require.Equal(t, uint64(0), ad.ID)
	require.Equal(t, "eea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364", fmt.Sprintf("%x", ad.Key))
}

func TestGrandpaAuthoritiesRawToAuthorities(t *testing.T) {
	ad := make([]*GrandpaAuthoritiesRaw, 2)
	buf := &bytes.Buffer{}
	data, _ := common.HexToBytes("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d714103640000000000000000b64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d7170100000000000000")
	buf.Write(data)

	authA, _ := common.HexToHash("0xeea1eabcac7d2c8a6459b7322cf997874482bfc3d2ec7a80888a3a7d71410364")
	authB, _ := common.HexToHash("0xb64994460e59b30364cad3c92e3df6052f9b0ebbb8f88460c194dc5794d6d717")

	expected := []*GrandpaAuthoritiesRaw{
		{Key: authA, ID: 0},
		{Key: authB, ID: 1},
	}

	var err error
	for i := range ad {
		ad[i], err = ad[i].Decode(buf)
		require.NoError(t, err)
	}

	require.Equal(t, expected, ad)
}
