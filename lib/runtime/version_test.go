package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionData(t *testing.T) {
	testAPIItem := &APIItem{
		Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Ver:  99,
	}

	version := NewVersionData(
		[]byte("polkadot"),
		[]byte("parity-polkadot"),
		0,
		25,
		0,
		[]*APIItem{testAPIItem},
		5,
	)

	b, err := version.Encode()
	require.NoError(t, err)

	dec := new(VersionData)
	err = dec.Decode(b)
	require.NoError(t, err)
	require.Equal(t, version, dec)
}

func TestLegacyVersionData(t *testing.T) {
	testAPIItem := &APIItem{
		Name: [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Ver:  99,
	}

	version := NewLegacyVersionData(
		[]byte("polkadot"),
		[]byte("parity-polkadot"),
		0,
		25,
		0,
		[]*APIItem{testAPIItem},
	)

	b, err := version.Encode()
	require.NoError(t, err)

	dec := new(LegacyVersionData)
	err = dec.Decode(b)
	require.NoError(t, err)
	require.Equal(t, version, dec)
}
