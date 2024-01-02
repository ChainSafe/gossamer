package crypto

// / Trait used for types that are really just a fixed-length array.
// pub trait ByteArray: AsRef<[u8]> + AsMut<[u8]> + for<'a> TryFrom<&'a [u8], Error = ()> {
type ByteArray interface {
	// 	/// The "length" of the values of this type, which is always the same.
	// 	const LEN: usize;

	/// A new instance from the given slice that should be `Self::LEN` bytes long.
	// 	fn from_slice(data: &[u8]) -> Result<Self, ()> {
	// 		Self::try_from(data)
	// 	}

	/// Return a `Vec<u8>` filled with raw data.
	// 	fn to_raw_vec(&self) -> Vec<u8> {
	// 		self.as_slice().to_vec()
	// 	}
	ToRawVec() []byte

	// /// Return a slice filled with raw data.
	//
	//	fn as_slice(&self) -> &[u8] {
	//		self.as_ref()
	//	}
}

// / Trait suitable for typical cryptographic key public type.
// pub trait Public: ByteArray + Derive + CryptoType + PartialEq + Eq + Clone + Send + Sync {}
type Public interface {
	ByteArray
}

// / An identifier for a specific cryptographic algorithm used by a key pair
// #[derive(Debug, Copy, Clone, Default, PartialEq, Eq, PartialOrd, Ord, Hash, Encode, Decode)]
// #[cfg_attr(feature = "std", derive(serde::Serialize, serde::Deserialize))]
// pub struct CryptoTypeId(pub [u8; 4]);
type CryptoTypeID string

// / An identifier for a type of cryptographic key.
// /
// / To avoid clashes with other modules when distributing your module publicly, register your
// / `KeyTypeId` on the list here by making a PR.
// /
// / Values whose first character is `_` are reserved for private use and won't conflict with any
// / public modules.
type KeyTypeID string

// / Known key types; this also functions as a global registry of key types for projects wishing to
// / avoid collisions with each other.
// /
// / It's not universal in the sense that *all* key types need to be mentioned here, it's just a
// / handy place to put common key types.
const (
	/// Key type for Babe module, built-in. Identified as `babe`.
	BABE KeyTypeID = "babe"
	/// Key type for Grandpa module, built-in. Identified as `gran`.
	GRANDPA KeyTypeID = "gran"
	/// Key type for controlling an account in a Substrate runtime, built-in. Identified as `acco`.
	Account KeyTypeID = "acco"
	/// Key type for Aura module, built-in. Identified as `aura`.
	AURA KeyTypeID = "aura"
	/// Key type for ImOnline module, built-in. Identified as `imon`.
	ImOnline KeyTypeID = "imon"
	/// Key type for AuthorityDiscovery module, built-in. Identified as `audi`.
	AuthorityDiscovery KeyTypeID = "audi"
	/// Key type for staking, built-in. Identified as `stak`.
	Staking KeyTypeID = "stak"
	/// A key type for signing statements
	Statement KeyTypeID = "stmt"
	/// A key type ID useful for tests.
	Dummy KeyTypeID = "dumy"
)
