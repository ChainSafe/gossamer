package keystore

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
)

// / KeyStore generates, stores and provides access to secret keys.
type KeyStore interface {
	/// Generate an ed25519 signature for a given message.
	///
	/// Receives [`KeyTypeId`] and an [`ed25519::Public`] key to be able to map
	/// them to a private key that exists in the keystore.
	///
	/// Returns an [`ed25519::Signature`] or `None` in case the given `key_type`
	/// and `public` combination doesn't exist in the keystore.
	/// An `Err` will be returned if generating the signature itself failed.
	Ed25519Sign(keyType crypto.KeyTypeID, public ed25519.Public, msg []byte) (*ed25519.Signature, error)
}
