package peerid

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestNewRandomPeerID(t *testing.T) {
	require.NotEmpty(t, NewRandomPeerID().ID)
}

func TestFromED25519(t *testing.T) {
	_, public, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)
	original, err := peer.IDFromPublicKey(public)
	require.NoError(t, err)
	t.Logf("%v", original)

	peerID, err := NewPeerID([]byte(original))
	require.NoError(t, err)
	require.Equal(t, original, peerID.ID)

	key := peerID.Ed25519()
	require.NotNil(t, key)
	from := NewPeerIDFromEd25519(*key)
	require.NotNil(t, from)
	require.Equal(t, original, from.ID)
}
