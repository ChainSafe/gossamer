package common

var (
	// CodeKey is the key where runtime code is stored in the trie
	CodeKey = []byte(":code")

	// UpgradedToDualRefKey is set to true (0x01) if the account format has been upgraded to v0.9
	// it's set to empty or false (0x00) otherwise
	UpgradedToDualRefKey = MustHexToBytes("0x26aa394eea5630e07c48ae0c9558cef7c21aab032aaa6e946ca50ad39ab66603")
)

// BalanceKey returns the storage trie key for the balance of the account with the given public key
// TODO: deprecate
func BalanceKey(key [32]byte) ([]byte, error) {
	accKey := append([]byte("balance:"), key[:]...)

	hash, err := Blake2bHash(accKey)
	if err != nil {
		return nil, err
	}

	return hash[:], nil
}

// NonceKey returns the storage trie key for the nonce of the account with the given public key
// TODO: deprecate
func NonceKey(key [32]byte) ([]byte, error) {
	accKey := append([]byte("nonce:"), key[:]...)

	hash, err := Blake2bHash(accKey)
	if err != nil {
		return nil, err
	}

	return hash[:], nil
}
