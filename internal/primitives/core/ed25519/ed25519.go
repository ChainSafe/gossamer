package ed25519

import (
	gocrypto "crypto"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ChainSafe/go-schnorrkel"
	"github.com/ChainSafe/gossamer/internal/primitives/core/crypto"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hashing"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// / A secret seed (which is bytewise essentially equivalent to a SecretKey).
// /
// / We need it as a different type because `Seed` is expected to be AsRef<[u8]>.
// #[cfg(feature = "full_crypto")]
// type Seed = [u8; 32];
type seed [32]byte

// / A public key.
type Public [32]byte

// / Derive a single hard junction.
// #[cfg(feature = "full_crypto")]
//
//	fn derive_hard_junction(secret_seed: &Seed, cc: &[u8; 32]) -> Seed {
//		("Ed25519HDKD", secret_seed, cc).using_encoded(sp_core_hashing::blake2_256)
//	}
func deriveHardJunction(secretSeed seed, cc [32]byte) seed {
	tuple := struct {
		ID         string
		SecretSeed seed
		CC         [32]byte
	}{"Ed25519HDKD", secretSeed, cc}
	encoded := scale.MustMarshal(tuple)
	return hashing.Blake2_256(encoded)
}

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

// / Derive a child key from a series of given junctions.
// fn derive<Iter: Iterator<Item = DeriveJunction>>(
//
//	&self,
//	path: Iter,
//	seed: Option<Self::Seed>,
//
// ) -> Result<(Self, Option<Self::Seed>), DeriveError>;
func (p Pair) Derive(path []crypto.DeriveJunction, seed *[32]byte) (crypto.Pair[[32]byte], [32]byte, error) {
	var acc [32]byte
	copy(acc[:], p.secret.Seed())
	for _, j := range path {
		switch cc := j.Value().(type) {
		case crypto.DeriveJunctionSoft:
			return Pair{}, [32]byte{}, fmt.Errorf("soft key in path")
		case crypto.DeriveJunctionHard:
			acc = deriveHardJunction(acc, cc)
		}
	}
	pair := NewPairFromSeed(acc)
	return pair, acc, nil
}

// / Get the seed for this key.
//
//	pub fn seed(&self) -> Seed {
//		self.secret.into()
//	}
func (p Pair) Seed() [32]byte {
	var seed [32]byte
	copy(seed[:], p.secret.Seed())
	return seed
}

// / Generate new key pair from the provided `seed`.
// /
// / @WARNING: THIS WILL ONLY BE SECURE IF THE `seed` IS SECURE. If it can be guessed
// / by an attacker then they can also derive your key.
//
//	fn from_seed(seed: &Self::Seed) -> Self {
//		Self::from_seed_slice(seed.as_ref()).expect("seed has valid length; qed")
//	}
func NewPairFromSeed(seed [32]byte) Pair {
	return NewPairFromSeedSlice(seed[:])
}

// / Make a new key pair from secret seed material. The slice must be the correct size or
// / it will return `None`.
// /
// / @WARNING: THIS WILL ONLY BE SECURE IF THE `seed` IS SECURE. If it can be guessed
// / by an attacker then they can also derive your key.
func NewPairFromSeedSlice(seedSlice []byte) Pair {
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
func NewPairFromPhrase(phrase string, password *string) (pair Pair, seed [32]byte, err error) {
	pass := ""
	if password != nil {
		pass = *password
	}
	bigSeed, err := schnorrkel.SeedFromMnemonic(phrase, pass)
	if err != nil {
		return Pair{}, [32]byte{}, err
	}

	if !(32 <= len(bigSeed)) {
		panic("huh?")
	}

	seedSlice := bigSeed[:][0:32]
	copy(seed[:], seedSlice)
	return NewPairFromSeedSlice(seedSlice), seed, nil
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

func NewPairFromStringWithSeed(s string, passwordOverride *string) (pair crypto.Pair[[32]byte], seed [32]byte, err error) {
	sURI, err := crypto.NewSecretURI(s)
	if err != nil {
		return Pair{}, [32]byte{}, err
	}
	var password *string
	if passwordOverride != nil {
		password = passwordOverride
	} else {
		password = sURI.Password
	}

	var (
		root Pair
		// seed []byte
	)
	trimmedPhrase := strings.TrimPrefix(sURI.Phrase, "0x")
	if trimmedPhrase != sURI.Phrase {
		seedBytes, err := hex.DecodeString(trimmedPhrase)
		if err != nil {
			return Pair{}, [32]byte{}, err
		}
		root = NewPairFromSeedSlice(seedBytes)
		copy(seed[:], seedBytes)
	} else {
		root, seed, err = NewPairFromPhrase(sURI.Phrase, password)
		if err != nil {
			return Pair{}, [32]byte{}, err
		}
	}
	return root.Derive(sURI.Junctions, &seed)
}

// / Interprets the string `s` in order to generate a key pair.
// /
// / See [`from_string_with_seed`](Pair::from_string_with_seed) for more extensive documentation.
func NewPairFromString(s string, passwordOverride *string) (crypto.Pair[[32]byte], error) {
	pair, _, err := NewPairFromStringWithSeed(s, passwordOverride)
	return pair, err
}

var _ crypto.Pair[[32]byte] = Pair{}

// / A signature (a 512-bit value).
type Signature [64]byte
