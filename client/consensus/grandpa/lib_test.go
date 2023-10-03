package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	finalityGrandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_checkMessageSignature(t *testing.T) {
	kp, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	// TODO Empty value until message is exported
	message := finalityGrandpa.Message[string, uint]{}

	msg := messageData[string, uint]{
		1,
		2,
		message,
	}

	encMsg, err := scale.Marshal(msg)
	require.NoError(t, err)

	sig, err := kp.Sign(encMsg)
	require.NoError(t, err)

	valid, err := checkMessageSignature[string, uint](message, kp.Public().(*ed25519.PublicKey), sig, 1, 2)
	require.NoError(t, err)
	require.True(t, valid)
}
