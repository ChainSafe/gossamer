package trie

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTrie(t *testing.T) *Trie {
	t.Helper()

	tr := NewEmptyTrie()

	tr.Put([]byte("transaction.data"), []byte("some-value"))
	tr.Put([]byte("transaction.sender"), []byte("some-sender"))
	tr.Put([]byte("xcm-message"), []byte("some-message"))

	return tr
}

func TestVerifyProof(t *testing.T) {
	tr := setupTrie(t)

	value, err := VerifyProof(tr, []byte("transaction.data"))
	require.NoError(t, err)
	require.Equal(t, value, []byte("some-value"))
}

func TestVerifyProof_KeyDoesntValid(t *testing.T) {
	tr := setupTrie(t)

	value, err := VerifyProof(tr, []byte("another-key"))
	require.Error(t, err, ErrInvalidProof)
	require.Nil(t, value)
}

func TestVerifyProof_EmptyInputs(t *testing.T) {
	tr := NewEmptyTrie()
	value, err := VerifyProof(tr, []byte("transaction.data"))
	require.Error(t, err, "cannot verify proof of an empty")
	require.Nil(t, value)

	tr = setupTrie(t)

	value, err = VerifyProof(tr, []byte{})
	require.Error(t, err, "cannot verify proof of an empty key")
	require.Nil(t, value)

}
