package runtime

import (
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/stretchr/testify/require"
)

func TestBackgroundSignVerification(t *testing.T) {
	signs := generateEd25519Signatures(t, 2)
	signVerify := new(SignatureVerifier)

	go signVerify.Start()
	time.Sleep(1 * time.Second)

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	// Wait for background go routine to verify signature.
	time.Sleep(1 * time.Second)
	require.True(t, signVerify.IsEmpty())
	require.False(t, signVerify.IsInValid())
}

func TestInvalidSignatureBatch(t *testing.T) {
	signs := generateEd25519Signatures(t, 2)

	msg := []byte("ed25519")
	key, err := ed25519.GenerateKeypair()
	require.NoError(t, err)

	// Invalid Sign
	sigData, err := common.HexToBytes("0x90f27b8b488db00b00606796d2987f6a5f59ae62ea05effe84fef5b8b0e549984a691139ad57a3f0b906637673aa2f63d1f55cb1a69199d4009eea23ceaddc9301")
	require.Nil(t, err)

	signature := &Signature{
		PubKey:    key.Public().Encode(),
		Sign:      sigData,
		Msg:       msg,
		KyeTypeID: crypto.Ed25519Type,
	}

	signs = append(signs, signature)

	signVerify := new(SignatureVerifier)
	go signVerify.Start()
	time.Sleep(1 * time.Second)

	for _, sig := range signs {
		signVerify.Add(sig)
	}
	require.False(t, signVerify.Finish())
}

func TestValidSignatureBatch(t *testing.T) {
	signs := generateEd25519Signatures(t, 2)
	signVerify := new(SignatureVerifier)

	go signVerify.Start()
	time.Sleep(1 * time.Second)

	for _, sig := range signs {
		signVerify.Add(sig)
	}

	require.True(t, signVerify.Finish())
}

func TestAllCryptoTypeSignature(t *testing.T) {
	edSignatures := generateEd25519Signatures(t, 1)

	srMsg := []byte("sr25519")
	srKey, err := sr25519.GenerateKeypair()
	require.NoError(t, err)

	srSig, err := srKey.Private().Sign(srMsg)
	require.NoError(t, err)

	srSignature := &Signature{
		PubKey:    srKey.Public().Encode(),
		Sign:      srSig,
		Msg:       srMsg,
		KyeTypeID: crypto.Sr25519Type,
	}

	blakeHash, err := common.Blake2bHash([]byte("secp256k1"))
	require.NoError(t, err)

	secpPubKey, secpPrvKey := utils.GenerateKeyPairs()
	secpSigData, err := secp256k1.Sign(blakeHash.ToBytes(), secpPrvKey)
	require.NoError(t, err)

	secpSigData = secpSigData[:len(secpSigData)-1] // remove recovery id
	secpSignature := &Signature{
		PubKey:    secpPubKey,
		Sign:      secpSigData,
		Msg:       blakeHash.ToBytes(),
		KyeTypeID: crypto.Secp256k1Type,
	}

	signVerify := new(SignatureVerifier)

	go signVerify.Start()
	time.Sleep(1 * time.Second)

	signVerify.Add(edSignatures[0])
	signVerify.Add(srSignature)
	signVerify.Add(secpSignature)

	require.True(t, signVerify.Finish())
}
