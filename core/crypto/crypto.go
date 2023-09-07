package crypto

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
