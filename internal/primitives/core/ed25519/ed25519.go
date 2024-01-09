package ed25519

import (
	gocrypto "crypto"
	"crypto/ed25519"
	"encoding/hex"
	"strings"

	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/tyler-smith/go-bip39"
)

// / A public key.
type Public [32]byte

// / A key pair.
// #[cfg(feature = "full_crypto")]
// #[derive(Copy, Clone)]
// pub struct Pair {
type Pair struct {
	// public: VerificationKey,
	public gocrypto.PublicKey
	// secret: SigningKey,
	secret ed25519.PrivateKey
}

// / Generate new key pair from the provided `seed`.
// /
// / @WARNING: THIS WILL ONLY BE SECURE IF THE `seed` IS SECURE. If it can be guessed
// / by an attacker then they can also derive your key.
func NewPairFromSeed(seedSlice []byte) Pair {
	secret := ed25519.NewKeyFromSeed(seedSlice)
	public := secret.Public()
	return Pair{
		public: public,
		secret: secret,
	}
}

// /// Returns the KeyPair from the English BIP39 seed `phrase`, or `None` if it's invalid.
// #[cfg(feature = "std")]
// fn from_phrase(
//
//	phrase: &str,
//	password: Option<&str>,
//
//	) -> Result<(Self, Self::Seed), SecretStringError> {
//		let mnemonic = Mnemonic::from_phrase(phrase, Language::English)
//			.map_err(|_| SecretStringError::InvalidPhrase)?;
//		let big_seed =
//			substrate_bip39::seed_from_entropy(mnemonic.entropy(), password.unwrap_or(""))
//				.map_err(|_| SecretStringError::InvalidSeed)?;
//		let mut seed = Self::Seed::default();
//		let seed_slice = seed.as_mut();
//		let seed_len = seed_slice.len();
//		debug_assert!(seed_len <= big_seed.len());
//		seed_slice[..seed_len].copy_from_slice(&big_seed[..seed_len]);
//		Self::from_seed_slice(seed_slice).map(|x| (x, seed))
//	}
func NewPairFromPhrase(phrase string, password *string) (pair Pair, seed []byte, err error) {
	entropy, err := bip39.EntropyFromMnemonic(phrase)
	if err != nil {
		return Pair{}, nil, err
	}
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

func NewPairFromStringWithSeed(s string, passwordOverride *string) (pair Pair, seed []byte, err error) {
	sURI, err := crypto.NewSecretURI(s)
	if err != nil {
		return Pair{}, nil, err
	}
	var password *string
	if passwordOverride != nil {
		password = passwordOverride
	} else {
		password = sURI.Password
	}

	var (
		root Pair
		seed []byte
	)
	trimmedPhrase := strings.TrimPrefix(sURI.Phrase, "0x")
	if trimmedPhrase != sURI.Phrase {
		seedBytes, err := hex.DecodeString(trimmedPhrase)
		if err != nil {
			return Pair{}, nil, err
		}
		root = NewPairFromSeed(seedBytes)
		seed = seedBytes
	} else {
		NewPairFromPhrase()
	}

	return Pair{}, nil, nil
}

// / A signature (a 512-bit value).
type Signature [64]byte
