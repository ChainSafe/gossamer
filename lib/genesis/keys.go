// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

const (
	// babePrefixHex is the hex encoding of: Twox128Hash("babe")
	babePrefixHex = "0x014f204c006a2837deb5551ba5211d6c"
	// BABEAuthoritiesKeyHex is the hex encoding of:
	// Twox128Hash("babe") + Twox128Hash("Authorities")
	BABEAuthoritiesKeyHex = babePrefixHex + "5e0621c4869aa60c02be9adcc98a0d1d"
	// BABERandomnessKeyHex is the hex encoding of:
	// Twox128Hash("babe") + Twox128Hash("Randomness")
	BABERandomnessKeyHex = babePrefixHex + "7a414cb008e0e61e46722aa60abdd672"

	// GrandpaAuthoritiesKeyHex is the hex encoding of the key to the GRANDPA
	// authority data in the storage trie.
	GrandpaAuthoritiesKeyHex = "0x3a6772616e6470615f617574686f726974696573"

	// systemPrefixHex is the hex encoding of: Twox128Hash("System")
	systemPrefixHex = "0x26aa394eea5630e07c48ae0c9558cef7"
	// systemAccountKeyHex is the hex encoding of:
	// Twox128Hash("babe") + Twox128Hash("Account")
	systemAccountKeyHex = systemPrefixHex + "b99d880ec681799c0cf30e8886371da9"
)
