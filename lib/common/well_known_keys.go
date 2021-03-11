package common

var (
	// CodeKey is the key where runtime code is stored in the trie
	CodeKey = []byte(":code")
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

// StorageKey returns the storage key for the given module and variable name
func StorageKey(module, name string) ([]byte, error) {
	h0, err := Twox128Hash([]byte(module))
	if err != nil {
		return nil, err
	}
	h1, err := Twox128Hash([]byte(name))
	if err != nil {
		return nil, err
	}
	return append(h0, h1...), nil
}
