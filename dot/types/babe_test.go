package types

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"

	"github.com/stretchr/testify/require"
)

func TestBABEAuthorityRaw(t *testing.T) {
	exp := []byte{0, 91, 50, 25, 214, 94, 119, 36, 71, 216, 33, 152, 85, 184, 34, 120, 61, 161, 164, 223, 76, 53, 40, 246, 76, 38, 235, 204, 43, 31, 179, 28, 1, 0, 0, 0, 0, 0, 0, 0}

	dec := AuthorityRaw{}
	err := scale.Unmarshal(exp, &dec)
	require.NoError(t, err)

	enc, err := scale.Marshal(dec)
	require.NoError(t, err)
	require.Equal(t, exp, enc)
}

func TestBABEAuthority(t *testing.T) {
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	ad := NewAuthority(kr.Alice().Public().(*sr25519.PublicKey), 77)
	enc, _ := ad.Encode()

	buf := &bytes.Buffer{}
	buf.Write(enc)

	res := new(Authority)
	err = res.DecodeSr25519(buf)
	require.NoError(t, err)
	require.Equal(t, res.Key.Encode(), ad.Key.Encode())
	require.Equal(t, res.Weight, ad.Weight)
}

func TestBABEAuthorities_ToRaw(t *testing.T) {
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	ad := NewAuthority(kr.Alice().Public().(*sr25519.PublicKey), 77)
	raw := ad.ToRaw()

	res := new(Authority)
	err = res.FromRawSr25519(raw)
	require.NoError(t, err)
	require.Equal(t, res.Key.Encode(), ad.Key.Encode())
	require.Equal(t, res.Weight, ad.Weight)
}

func TestEpochData(t *testing.T) {
	kr, err := keystore.NewSr25519Keyring()
	require.NoError(t, err)

	auth := Authority{
		Key:    kr.Alice().Public().(*sr25519.PublicKey),
		Weight: 1,
	}

	data := &EpochData{
		Authorities: []Authority{auth},
		Randomness:  [32]byte{77},
	}

	raw := data.ToEpochDataRaw()
	unraw, err := raw.ToEpochData()
	require.NoError(t, err)
	require.Equal(t, data.Randomness, unraw.Randomness)

	for i, auth := range unraw.Authorities {
		expected, err := data.Authorities[i].Encode()
		require.NoError(t, err)
		res, err := auth.Encode()
		require.NoError(t, err)
		require.Equal(t, expected, res)
	}
}
