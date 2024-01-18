package crypto

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hashing"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// /// The root phrase for our publicly known keys.
// pub const DEV_PHRASE: &str =
//
//	"bottom drive obey lake curtain smoke basket hold race lonely fit walk";
const DevPhrase = "bottom drive obey lake curtain smoke basket hold race lonely fit walk"

// / A since derivation junction description. It is the single parameter used when creating
// / a new secret key from an existing secret key and, in the case of `SoftRaw` and `SoftIndex`
// / a new public key from an existing public key.
// #[derive(Copy, Clone, Eq, PartialEq, Hash, Debug, Encode, Decode)]
// #[cfg(feature = "full_crypto")]
// pub enum DeriveJunction {
type DeriveJunction struct {
	inner any
}
type DeriveJunctions interface {
	DeriveJunctionSoft | DeriveJunctionHard
}

func (dj DeriveJunction) Value() any {
	if dj.inner == nil {
		panic("nil inner for DeriveJunction")
	}
	return dj.inner
}

// /// Soft (vanilla) derivation. Public keys have a correspondent derivation.
// Soft([u8; JUNCTION_ID_LEN]),
type DeriveJunctionSoft [32]byte

//		/// Hard ("hardened") derivation. Public keys do not have a correspondent derivation.
//		Hard([u8; JUNCTION_ID_LEN]),
//	}
type DeriveJunctionHard [32]byte

// #[cfg(feature = "full_crypto")]
// impl DeriveJunction {
// 	/// Consume self to return a soft derive junction with the same chain code.
// 	pub fn soften(self) -> Self {
// 		DeriveJunction::Soft(self.unwrap_inner())
// 	}

// /// Consume self to return a hard derive junction with the same chain code.
//
//	pub fn harden(self) -> Self {
//		DeriveJunction::Hard(self.unwrap_inner())
//	}
func (dj *DeriveJunction) Harden() DeriveJunction {
	switch inner := dj.inner.(type) {
	case DeriveJunctionSoft:
		dj.inner = DeriveJunctionHard(inner)
	}
	return *dj
}

// 	/// Consume self to return the chain code.
// 	pub fn unwrap_inner(self) -> [u8; JUNCTION_ID_LEN] {
// 		match self {
// 			DeriveJunction::Hard(c) | DeriveJunction::Soft(c) => c,
// 		}
// 	}

// 	/// Get a reference to the inner junction id.
// 	pub fn inner(&self) -> &[u8; JUNCTION_ID_LEN] {
// 		match self {
// 			DeriveJunction::Hard(ref c) | DeriveJunction::Soft(ref c) => c,
// 		}
// 	}

// 	/// Return `true` if the junction is soft.
// 	pub fn is_soft(&self) -> bool {
// 		matches!(*self, DeriveJunction::Soft(_))
// 	}

// 	/// Return `true` if the junction is hard.
// 	pub fn is_hard(&self) -> bool {
// 		matches!(*self, DeriveJunction::Hard(_))
// 	}

// /// Create a new soft (vanilla) DeriveJunction from a given, encodable, value.
// ///
// /// If you need a hard junction, use `hard()`.
//
//	pub fn soft<T: Encode>(index: T) -> Self {
//		let mut cc: [u8; JUNCTION_ID_LEN] = Default::default();
//		index.using_encoded(|data| {
//			if data.len() > JUNCTION_ID_LEN {
//				cc.copy_from_slice(&sp_core_hashing::blake2_256(data));
//			} else {
//				cc[0..data.len()].copy_from_slice(data);
//			}
//		});
//		DeriveJunction::Soft(cc)
//	}
func NewDeriveJunctionSoft(index any) (DeriveJunctionSoft, error) {
	var cc = [32]byte{}
	data, err := scale.Marshal(index)
	if err != nil {
		return DeriveJunctionSoft{}, err
	}

	if len(data) > 32 {
		cc = hashing.Blake2_256(data)
	} else {
		copy(cc[:], data)
	}
	return DeriveJunctionSoft(cc), nil
}

// 	/// Create a new hard (hardened) DeriveJunction from a given, encodable, value.
// 	///
// 	/// If you need a soft junction, use `soft()`.
// 	pub fn hard<T: Encode>(index: T) -> Self {
// 		Self::soft(index).harden()
// 	}
// }

// #[cfg(feature = "full_crypto")]
// impl<T: AsRef<str>> From<T> for DeriveJunction {
// 	fn from(j: T) -> DeriveJunction {
// 		let j = j.as_ref();
// 		let (code, hard) =
// 			if let Some(stripped) = j.strip_prefix('/') { (stripped, true) } else { (j, false) };

// 		let res = if let Ok(n) = str::parse::<u64>(code) {
// 			// number
// 			DeriveJunction::soft(n)
// 		} else {
// 			// something else
// 			DeriveJunction::soft(code)
// 		};

//			if hard {
//				res.harden()
//			} else {
//				res
//			}
//		}
//	}
func NewDeriveJunctionFromString(j string) DeriveJunction {
	hard := false
	trimmed := strings.TrimPrefix(j, "/")
	if trimmed != j {
		hard = true
	}
	code := trimmed

	var res DeriveJunction
	n, err := strconv.Atoi(code)
	if err != nil {
		soft, err := NewDeriveJunctionSoft(n)
		if err != nil {
			panic(err)
		}
		res = DeriveJunction{
			inner: soft,
		}
	} else {
		soft, err := NewDeriveJunctionSoft(code)
		if err != nil {
			panic(err)
		}
		res = DeriveJunction{
			inner: soft,
		}
	}

	if hard {
		return res.Harden()
	} else {
		return res
	}
}

func NewDeriveJunction[V DeriveJunctions](value V) DeriveJunction {
	return DeriveJunction{
		inner: value,
	}
}

// static ref SECRET_PHRASE_REGEX: Regex = Regex::new(r"^(?P<phrase>[\d\w ]+)?(?P<path>(//?[^/]+)*)(///(?P<password>.*))?$")
//
//	.expect("constructed from known-good static value; qed");
var secretPhraseRegex = regexp.MustCompile(`^(?P<phrase>[\d\w ]+)?(?P<path>(//?[^/]+)*)(///(?P<password>.*))?$`)

// static ref  JUNCTION_REGEX: Regex = Regex::new(r"/(/?[^/]+)")
//
//	.expect("constructed from known-good static value; qed");]
var junctionRegex = regexp.MustCompile(`/(/?[^/]+)`)

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
type Public[Signature any] interface {
	ByteArray

	/// Verify a signature on a message. Returns true if the signature is good.
	// fn verify<M: AsRef<[u8]>>(sig: &Self::Signature, message: M, pubkey: &Self::Public) -> bool;
	Verify(sig Signature, message []byte) bool
}

// / A secret uri (`SURI`) that can be used to generate a key pair.
// /
// / The `SURI` can be parsed from a string. The string is interpreted in the following way:
// /
// / - If `string` is a possibly `0x` prefixed 64-digit hex string, then it will be interpreted
// / directly as a `MiniSecretKey` (aka "seed" in `subkey`).
// / - If `string` is a valid BIP-39 key phrase of 12, 15, 18, 21 or 24 words, then the key will
// / be derived from it. In this case:
// /   - the phrase may be followed by one or more items delimited by `/` characters.
// /   - the path may be followed by `///`, in which case everything after the `///` is treated
// / as a password.
// / - If `string` begins with a `/` character it is prefixed with the Substrate public `DEV_PHRASE`
// /   and interpreted as above.
// /
// / In this case they are interpreted as HDKD junctions; purely numeric items are interpreted as
// / integers, non-numeric items as strings. Junctions prefixed with `/` are interpreted as soft
// / junctions, and with `//` as hard junctions.
// /
// / There is no correspondence mapping between `SURI` strings and the keys they represent.
// / Two different non-identical strings can actually lead to the same secret being derived.
// / Notably, integer junction indices may be legally prefixed with arbitrary number of zeros.
// / Similarly an empty password (ending the `SURI` with `///`) is perfectly valid and will
// / generally be equivalent to no password at all.
// /
// / # Example
// /
// / Parse [`DEV_PHRASE`] secret uri with junction:
// /
// / ```
// / # use sp_core::crypto::{SecretUri, DeriveJunction, DEV_PHRASE, ExposeSecret};
// / # use std::str::FromStr;
// / let suri = SecretUri::from_str("//Alice").expect("Parse SURI");
// /
// / assert_eq!(vec![DeriveJunction::from("Alice").harden()], suri.junctions);
// / assert_eq!(DEV_PHRASE, suri.phrase.expose_secret());
// / assert!(suri.password.is_none());
// / ```
// /
// / Parse [`DEV_PHRASE`] secret ui with junction and password:
// /
// / ```
// / # use sp_core::crypto::{SecretUri, DeriveJunction, DEV_PHRASE, ExposeSecret};
// / # use std::str::FromStr;
// / let suri = SecretUri::from_str("//Alice///SECRET_PASSWORD").expect("Parse SURI");
// /
// / assert_eq!(vec![DeriveJunction::from("Alice").harden()], suri.junctions);
// / assert_eq!(DEV_PHRASE, suri.phrase.expose_secret());
// / assert_eq!("SECRET_PASSWORD", suri.password.unwrap().expose_secret());
// / ```
// /
// / Parse [`DEV_PHRASE`] secret ui with hex phrase and junction:
// /
// / ```
// / # use sp_core::crypto::{SecretUri, DeriveJunction, DEV_PHRASE, ExposeSecret};
// / # use std::str::FromStr;
// / let suri = SecretUri::from_str("0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a//Alice").expect("Parse SURI");
// /
// / assert_eq!(vec![DeriveJunction::from("Alice").harden()], suri.junctions);
// / assert_eq!("0xe5be9a5092b81bca64be81d212e7f2f9eba183bb7a90954f7b76361f6edb5c0a", suri.phrase.expose_secret());
// / assert!(suri.password.is_none());
// / ```
type SecretURI struct {
	/// The phrase to derive the private key.
	///
	/// This can either be a 64-bit hex string or a BIP-39 key phrase.
	// pub phrase: SecretString,
	Phrase string
	/// Optional password as given as part of the uri.
	// pub password: Option<SecretString>,
	Password *string
	/// The junctions as part of the uri.
	// pub junctions: Vec<DeriveJunction>,
	Junctions []DeriveJunction
}

// impl sp_std::str::FromStr for SecretUri {
// 	type Err = SecretStringError;

// 	fn from_str(s: &str) -> Result<Self, Self::Err> {
// 		let cap = SECRET_PHRASE_REGEX.captures(s).ok_or(SecretStringError::InvalidFormat)?;

// 		let junctions = JUNCTION_REGEX
// 			.captures_iter(&cap["path"])
// 			.map(|f| DeriveJunction::from(&f[1]))
// 			.collect::<Vec<_>>();

// 		let phrase = cap.name("phrase").map(|r| r.as_str()).unwrap_or(DEV_PHRASE);
// 		let password = cap.name("password");

//			Ok(Self {
//				phrase: SecretString::from_str(phrase).expect("Returns infallible error; qed"),
//				password: password.map(|v| {
//					SecretString::from_str(v.as_str()).expect("Returns infallible error; qed")
//				}),
//				junctions,
//			})
//		}
//	}
func NewSecretURI(s string) (SecretURI, error) {
	matches := secretPhraseRegex.FindStringSubmatch(s)
	if matches == nil {
		return SecretURI{}, fmt.Errorf("invalid format")
	}

	var (
		junctions []DeriveJunction
		phrase    = DevPhrase
		password  *string
	)
	for i, name := range secretPhraseRegex.SubexpNames() {
		if i == 0 {
			continue
		}
		switch name {
		case "path":
			junctionMatches := junctionRegex.FindAllString(matches[i], -1)
			for _, jm := range junctionMatches {
				junctions = append(junctions, NewDeriveJunctionFromString(jm))
			}
		case "phrase":
			if matches[i] != "" {
				phrase = matches[i]
			}
		case "password":
			if matches[i] != "" {
				pw := matches[i]
				password = &pw
			}
		}
	}
	return SecretURI{
		Phrase:    phrase,
		Password:  password,
		Junctions: junctions,
	}, nil
}

// / Trait suitable for typical cryptographic PKI key pair type.
// /
// / For now it just specifies how to create a key from a phrase and derivation path.
type Pair[Seed, Signature any] interface {
	// /// The type which is used to encode a public key.
	// type Public: Public + Hash;

	// /// The type used to (minimally) encode the data required to securely create
	// /// a new key pair.
	// type Seed: Default + AsRef<[u8]> + AsMut<[u8]> + Clone;

	// /// The type used to represent a signature. Can be created from a key pair and a message
	// /// and verified with the message and a public key.
	// type Signature: AsRef<[u8]>;

	// /// Generate new secure (random) key pair.
	// ///
	// /// This is only for ephemeral keys really, since you won't have access to the secret key
	// /// for storage. If you want a persistent key pair, use `generate_with_phrase` instead.
	// #[cfg(feature = "std")]
	// fn generate() -> (Self, Self::Seed) {
	// 	let mut seed = Self::Seed::default();
	// 	OsRng.fill_bytes(seed.as_mut());
	// 	(Self::from_seed(&seed), seed)
	// }

	// /// Generate new secure (random) key pair and provide the recovery phrase.
	// ///
	// /// You can recover the same key later with `from_phrase`.
	// ///
	// /// This is generally slower than `generate()`, so prefer that unless you need to persist
	// /// the key from the current session.
	// #[cfg(feature = "std")]
	// fn generate_with_phrase(password: Option<&str>) -> (Self, String, Self::Seed) {
	// 	let mnemonic = Mnemonic::new(MnemonicType::Words12, Language::English);
	// 	let phrase = mnemonic.phrase();
	// 	let (pair, seed) = Self::from_phrase(phrase, password)
	// 		.expect("All phrases generated by Mnemonic are valid; qed");
	// 	(pair, phrase.to_owned(), seed)
	// }

	// /// Returns the KeyPair from the English BIP39 seed `phrase`, or `None` if it's invalid.
	// #[cfg(feature = "std")]
	// fn from_phrase(
	// 	phrase: &str,
	// 	password: Option<&str>,
	// ) -> Result<(Self, Self::Seed), SecretStringError> {
	// 	let mnemonic = Mnemonic::from_phrase(phrase, Language::English)
	// 		.map_err(|_| SecretStringError::InvalidPhrase)?;
	// 	let big_seed =
	// 		substrate_bip39::seed_from_entropy(mnemonic.entropy(), password.unwrap_or(""))
	// 			.map_err(|_| SecretStringError::InvalidSeed)?;
	// 	let mut seed = Self::Seed::default();
	// 	let seed_slice = seed.as_mut();
	// 	let seed_len = seed_slice.len();
	// 	debug_assert!(seed_len <= big_seed.len());
	// 	seed_slice[..seed_len].copy_from_slice(&big_seed[..seed_len]);
	// 	Self::from_seed_slice(seed_slice).map(|x| (x, seed))
	// }

	// /// Derive a child key from a series of given junctions.
	// fn derive<Iter: Iterator<Item = DeriveJunction>>(
	// 	&self,
	// 	path: Iter,
	// 	seed: Option<Self::Seed>,
	// ) -> Result<(Self, Option<Self::Seed>), DeriveError>;
	Derive(path []DeriveJunction, seed *Seed) (Pair[Seed, Signature], Seed, error)

	// /// Generate new key pair from the provided `seed`.
	// ///
	// /// @WARNING: THIS WILL ONLY BE SECURE IF THE `seed` IS SECURE. If it can be guessed
	// /// by an attacker then they can also derive your key.
	// fn from_seed(seed: &Self::Seed) -> Self {
	// 	Self::from_seed_slice(seed.as_ref()).expect("seed has valid length; qed")
	// }

	// /// Make a new key pair from secret seed material. The slice must be the correct size or
	// /// it will return `None`.
	// ///
	// /// @WARNING: THIS WILL ONLY BE SECURE IF THE `seed` IS SECURE. If it can be guessed
	// /// by an attacker then they can also derive your key.
	// fn from_seed_slice(seed: &[u8]) -> Result<Self, SecretStringError>;

	/// Sign a message.
	// fn sign(&self, message: &[u8]) -> Self::Signature;
	Sign(message []byte) Signature

	// /// Verify a signature on a message. Returns true if the signature is good.
	// fn verify<M: AsRef<[u8]>>(sig: &Self::Signature, message: M, pubkey: &Self::Public) -> bool;

	/// Get the public key.
	// fn public(&self) -> Self::Public;
	Public() Public[Signature]

	// /// Return a vec filled with raw data.
	// fn to_raw_vec(&self) -> Vec<u8>;
}

// /// Interprets the string `s` in order to generate a key Pair. Returns both the pair and an
// /// optional seed, in the case that the pair can be expressed as a direct derivation from a seed
// /// (some cases, such as Sr25519 derivations with path components, cannot).
// ///
// /// This takes a helper function to do the key generation from a phrase, password and
// /// junction iterator.
// ///
// /// - If `s` is a possibly `0x` prefixed 64-digit hex string, then it will be interpreted
// /// directly as a `MiniSecretKey` (aka "seed" in `subkey`).
// /// - If `s` is a valid BIP-39 key phrase of 12, 15, 18, 21 or 24 words, then the key will
// /// be derived from it. In this case:
// ///   - the phrase may be followed by one or more items delimited by `/` characters.
// ///   - the path may be followed by `///`, in which case everything after the `///` is treated
// /// as a password.
// /// - If `s` begins with a `/` character it is prefixed with the Substrate public `DEV_PHRASE`
// ///   and
// /// interpreted as above.
// ///
// /// In this case they are interpreted as HDKD junctions; purely numeric items are interpreted as
// /// integers, non-numeric items as strings. Junctions prefixed with `/` are interpreted as soft
// /// junctions, and with `//` as hard junctions.
// ///
// /// There is no correspondence mapping between SURI strings and the keys they represent.
// /// Two different non-identical strings can actually lead to the same secret being derived.
// /// Notably, integer junction indices may be legally prefixed with arbitrary number of zeros.
// /// Similarly an empty password (ending the SURI with `///`) is perfectly valid and will
// /// generally be equivalent to no password at all.
// ///
// /// `None` is returned if no matches are found.
// #[cfg(feature = "std")]
// fn from_string_with_seed(
// 	s: &str,
// 	password_override: Option<&str>,
// ) -> Result<(Self, Option<Self::Seed>), SecretStringError> {
// 	use sp_std::str::FromStr;
// 	let SecretUri { junctions, phrase, password } = SecretUri::from_str(s)?;
// 	let password =
// 		password_override.or_else(|| password.as_ref().map(|p| p.expose_secret().as_str()));

// 	let (root, seed) = if let Some(stripped) = phrase.expose_secret().strip_prefix("0x") {
// 		array_bytes::hex2bytes(stripped)
// 			.ok()
// 			.and_then(|seed_vec| {
// 				let mut seed = Self::Seed::default();
// 				if seed.as_ref().len() == seed_vec.len() {
// 					seed.as_mut().copy_from_slice(&seed_vec);
// 					Some((Self::from_seed(&seed), seed))
// 				} else {
// 					None
// 				}
// 			})
// 			.ok_or(SecretStringError::InvalidSeed)?
// 	} else {
// 		Self::from_phrase(phrase.expose_secret().as_str(), password)
// 			.map_err(|_| SecretStringError::InvalidPhrase)?
// 	};
// 	root.derive(junctions.into_iter(), Some(seed))
// 		.map_err(|_| SecretStringError::InvalidPath)
// }

// /// Interprets the string `s` in order to generate a key pair.
// ///
// /// See [`from_string_with_seed`](Pair::from_string_with_seed) for more extensive documentation.
// #[cfg(feature = "std")]
// fn from_string(s: &str, password_override: Option<&str>) -> Result<Self, SecretStringError> {
// 	Self::from_string_with_seed(s, password_override).map(|x| x.0)
// }

// func NewPairFromStringWithSeed(s string, passwordOverride *string) (pair Pair, seed []byte, err error) {
// 	sURI, err := NewSecretURI(s)
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	var password *string
// 	if passwordOverride != nil {
// 		password = passwordOverride
// 	} else {
// 		password = sURI.Password
// 	}

// 	trimmedPhrase := strings.TrimPrefix(sURI.Phrase, "0x")
// 	if trimmedPhrase != sURI.Phrase {
// 		seedBytes, err := hex.DecodeString(trimmedPhrase)
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 	} else {

// 	}

// 	return nil, nil, nil
// }

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
