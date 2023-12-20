package applicationcrypto

// / A runtime interface for a public key.
type RuntimePublic[Signature any] interface {
	// pub trait RuntimePublic: Sized {
	/// The signature that will be generated when signing with the corresponding private key.
	// type Signature: Debug + Eq + PartialEq + Clone;

	/// Returns all public keys for the given key type in the keystore.
	// fn all(key_type: KeyTypeId) -> crate::Vec<Self>;

	/// Generate a public/private pair for the given key type with an optional `seed` and
	/// store it in the keystore.
	///
	/// The `seed` needs to be valid utf8.
	///
	/// Returns the generated public key.
	// fn generate_pair(key_type: KeyTypeId, seed: Option<Vec<u8>>) -> Self;

	/// Sign the given message with the corresponding private key of this public key.
	///
	/// The private key will be requested from the keystore using the given key type.
	///
	/// Returns the signature or `None` if the private key could not be found or some other error
	/// occurred.
	// fn sign<M: AsRef<[u8]>>(&self, key_type: KeyTypeId, msg: &M) -> Option<Self::Signature>;

	/// Verify that the given signature matches the given message using this public key.
	// fn verify<M: AsRef<[u8]>>(&self, msg: &M, signature: &Self::Signature) -> bool;
	Verify(msg []byte, signature Signature) bool

	/// Returns `Self` as raw vec.
	// fn to_raw_vec(&self) -> Vec<u8>;
}
