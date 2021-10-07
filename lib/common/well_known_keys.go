package common

var (
	// CodeKey is the key where runtime code is stored in the trie
	CodeKey = []byte(":code")

	// UpgradedToDualRefKey is set to true (0x01) if the account format has been upgraded to v0.9
	// it's set to empty or false (0x00) otherwise
	UpgradedToDualRefKey = MustHexToBytes("0x26aa394eea5630e07c48ae0c9558cef7c21aab032aaa6e946ca50ad39ab66603")
)
