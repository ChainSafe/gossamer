package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/keystore"

	"github.com/stretchr/testify/require"
)

func TestPubkeyToVoter(t *testing.T) {
	voters := newTestVoters(t)
	kr, err := keystore.NewEd25519Keyring()
	require.NoError(t, err)

	state := NewState(voters, 0, 0)
	voter, err := state.pubkeyToVoter(kr.Alice.Public().(*ed25519.PublicKey))
	require.NoError(t, err)
	require.Equal(t, voters[0], voter)
}
