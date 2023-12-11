package keystore

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/ecdsa"
	"github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/core/sr25519"
)

// / Something that generates, stores and provides access to secret keys.
type KeyStore interface {
	/// Returns all the sr25519 public keys for the given key type.
	Sr25519PublicKeys(keyType crypto.KeyTypeID) []sr25519.Public

	/// Generate a new sr25519 key pair for the given key type and an optional seed.
	///
	/// Returns an `sr25519::Public` key of the generated key pair or an `Err` if
	/// something failed during key generation.
	Sr25519GenerateNew(keyType crypto.KeyTypeID, seed *string) (sr25519.Public, error)

	/// Generate an sr25519 signature for a given message.
	///
	/// Receives [`KeyTypeId`] and an [`sr25519::Public`] key to be able to map
	/// them to a private key that exists in the keystore.
	///
	/// Returns an [`sr25519::Signature`] or `None` in case the given `key_type`
	/// and `public` combination doesn't exist in the keystore.
	/// An `Err` will be returned if generating the signature itself failed.
	Sr25519Sign(keyType crypto.KeyTypeID, public *sr25519.Public, msg []byte) (*sr25519.Signature, error)

	/// Generate an sr25519 VRF signature for the given data.
	///
	/// Receives [`KeyTypeId`] and an [`sr25519::Public`] key to be able to map
	/// them to a private key that exists in the keystore.
	///
	/// Returns `None` if the given `key_type` and `public` combination doesn't
	/// exist in the keystore or an `Err` when something failed.
	Sr25519VRFSign(keyType crypto.KeyTypeID, public sr25519.Public, data sr25519.VRFSignData) (*sr25519.VRFSignature, error)

	// /// Generate an sr25519 VRF output for a given input data.
	// ///
	// /// Receives [`KeyTypeId`] and an [`sr25519::Public`] key to be able to map
	// /// them to a private key that exists in the keystore.
	// ///
	// /// Returns `None` if the given `key_type` and `public` combination doesn't
	// /// exist in the keystore or an `Err` when something failed.
	// fn sr25519_vrf_output(
	// 	&self,
	// 	key_type: KeyTypeId,
	// 	public: &sr25519::Public,
	// 	input: &sr25519::vrf::VrfInput,
	// ) -> Result<Option<sr25519::vrf::VrfOutput>, Error>;
	Sr25519VRFOutput(keyType crypto.KeyTypeID, public sr25519.Public, input sr25519.VRFInput) (*sr25519.VRFOutput, error)

	// /// Returns all ed25519 public keys for the given key type.
	// fn ed25519_public_keys(&self, key_type: KeyTypeId) -> Vec<ed25519::Public>;
	Ed25519PublicKeys(keyType crypto.KeyTypeID) []ed25519.Public

	/// Generate a new ed25519 key pair for the given key type and an optional seed.
	///
	/// Returns an `ed25519::Public` key of the generated key pair or an `Err` if
	/// something failed during key generation.
	// fn ed25519_generate_new(
	// 	&self,
	// 	key_type: KeyTypeId,
	// 	seed: Option<&str>,
	// ) -> Result<ed25519::Public, Error>;
	Ed25519GenerateNew(keyType crypto.KeyTypeID, seed *string) (ed25519.Public, error)

	// /// Generate an ed25519 signature for a given message.
	// ///
	// /// Receives [`KeyTypeId`] and an [`ed25519::Public`] key to be able to map
	// /// them to a private key that exists in the keystore.
	// ///
	// /// Returns an [`ed25519::Signature`] or `None` in case the given `key_type`
	// /// and `public` combination doesn't exist in the keystore.
	// /// An `Err` will be returned if generating the signature itself failed.
	// fn ed25519_sign(
	// 	&self,
	// 	key_type: KeyTypeId,
	// 	public: &ed25519::Public,
	// 	msg: &[u8],
	// ) -> Result<Option<ed25519::Signature>, Error>;
	Ed25519Sign(keyType crypto.KeyTypeID, public ed25519.Public, msg []byte) (*ed25519.Signature, error)

	// /// Returns all ecdsa public keys for the given key type.
	// fn ecdsa_public_keys(&self, key_type: KeyTypeId) -> Vec<ecdsa::Public>;
	ECDSAPublicKeys(keyType crypto.KeyTypeID) []ecdsa.Public

	// /// Generate a new ecdsa key pair for the given key type and an optional seed.
	// ///
	// /// Returns an `ecdsa::Public` key of the generated key pair or an `Err` if
	// /// something failed during key generation.
	// fn ecdsa_generate_new(
	// 	&self,
	// 	key_type: KeyTypeId,
	// 	seed: Option<&str>,
	// ) -> Result<ecdsa::Public, Error>;
	ECDSAGenerateNew(keyType crypto.KeyTypeID, seed *string) (ecdsa.Public, error)

	// /// Generate an ecdsa signature for a given message.
	// ///
	// /// Receives [`KeyTypeId`] and an [`ecdsa::Public`] key to be able to map
	// /// them to a private key that exists in the keystore.
	// ///
	// /// Returns an [`ecdsa::Signature`] or `None` in case the given `key_type`
	// /// and `public` combination doesn't exist in the keystore.
	// /// An `Err` will be returned if generating the signature itself failed.
	// fn ecdsa_sign(
	// 	&self,
	// 	key_type: KeyTypeId,
	// 	public: &ecdsa::Public,
	// 	msg: &[u8],
	// ) -> Result<Option<ecdsa::Signature>, Error>;
	ECDSASign(keyType crypto.KeyTypeID, public ecdsa.Public, msg []byte) (*ecdsa.Signature, error)

	// /// Generate an ecdsa signature for a given pre-hashed message.
	// ///
	// /// Receives [`KeyTypeId`] and an [`ecdsa::Public`] key to be able to map
	// /// them to a private key that exists in the keystore.
	// ///
	// /// Returns an [`ecdsa::Signature`] or `None` in case the given `key_type`
	// /// and `public` combination doesn't exist in the keystore.
	// /// An `Err` will be returned if generating the signature itself failed.
	// fn ecdsa_sign_prehashed(
	// 	&self,
	// 	key_type: KeyTypeId,
	// 	public: &ecdsa::Public,
	// 	msg: &[u8; 32],
	// ) -> Result<Option<ecdsa::Signature>, Error>;
	ECDSASignPrehashed(keyType crypto.KeyTypeID, public ecdsa.Public, msg [32]byte) (*ecdsa.Signature, error)

	/// Insert a new secret key.
	// fn insert(&self, key_type: KeyTypeId, suri: &str, public: &[u8]) -> Result<(), ()>;

	/// List all supported keys of a given type.
	///
	/// Returns a set of public keys the signer supports in raw format.
	// fn keys(&self, key_type: KeyTypeId) -> Result<Vec<Vec<u8>>, Error>;

	/// Checks if the private keys for the given public key and key type combinations exist.
	///
	/// Returns `true` iff all private keys could be found.
	// fn has_keys(&self, public_keys: &[(Vec<u8>, KeyTypeId)]) -> bool;
	HasKeys(publicKeys []struct {
		Key []byte
		crypto.KeyTypeID
	}) bool

	/// Convenience method to sign a message using the given key type and a raw public key
	/// for secret lookup.
	///
	/// The message is signed using the cryptographic primitive specified by `crypto_id`.
	///
	/// Schemes supported by the default trait implementation:
	/// - sr25519
	/// - ed25519
	/// - ecdsa
	/// - bls381
	/// - bls377
	///
	/// To support more schemes you can overwrite this method.
	///
	/// Returns the SCALE encoded signature if key is found and supported, `None` if the key doesn't
	/// exist or an error when something failed.
	// fn sign_with(
	// 	&self,
	// 	id: KeyTypeId,
	// 	crypto_id: CryptoTypeId,
	// 	public: &[u8],
	// 	msg: &[u8],
	// ) -> Result<Option<Vec<u8>>, Error> {
	// 	use codec::Encode;

	// 	let signature = match crypto_id {
	// 		sr25519::CRYPTO_ID => {
	// 			let public = sr25519::Public::from_slice(public)
	// 				.map_err(|_| Error::ValidationError("Invalid public key format".into()))?;
	// 			self.sr25519_sign(id, &public, msg)?.map(|s| s.encode())
	// 		},
	// 		ed25519::CRYPTO_ID => {
	// 			let public = ed25519::Public::from_slice(public)
	// 				.map_err(|_| Error::ValidationError("Invalid public key format".into()))?;
	// 			self.ed25519_sign(id, &public, msg)?.map(|s| s.encode())
	// 		},
	// 		ecdsa::CRYPTO_ID => {
	// 			let public = ecdsa::Public::from_slice(public)
	// 				.map_err(|_| Error::ValidationError("Invalid public key format".into()))?;

	// 			self.ecdsa_sign(id, &public, msg)?.map(|s| s.encode())
	// 		},
	// 		#[cfg(feature = "bls-experimental")]
	// 		bls381::CRYPTO_ID => {
	// 			let public = bls381::Public::from_slice(public)
	// 				.map_err(|_| Error::ValidationError("Invalid public key format".into()))?;
	// 			self.bls381_sign(id, &public, msg)?.map(|s| s.encode())
	// 		},
	// 		#[cfg(feature = "bls-experimental")]
	// 		bls377::CRYPTO_ID => {
	// 			let public = bls377::Public::from_slice(public)
	// 				.map_err(|_| Error::ValidationError("Invalid public key format".into()))?;
	// 			self.bls377_sign(id, &public, msg)?.map(|s| s.encode())
	// 		},
	// 		_ => return Err(Error::KeyNotSupported(id)),
	// 	};
	// 	Ok(signature)
	// }
	SignWith(id crypto.KeyTypeID, cryptoID crypto.CryptoTypeID, public []byte, msg []byte) (*[]byte, error)
}
