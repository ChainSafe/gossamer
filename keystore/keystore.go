package keystore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/ChainSafe/gossamer/crypto"
	"golang.org/x/crypto/blake2b"
)

func GCMFromPassphrase(password []byte) (cipher.AEAD, error) {
	hash := blake2b.Sum256(password)

	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm, nil
}

func Encrypt(msg, password []byte) ([]byte, error) {
	gcm, err := GCMFromPassphrase(password)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		if err != nil {
			return nil, err
		}
	}

	ciphertext := gcm.Seal(nonce, nonce, msg, nil)
	return ciphertext, nil
}

func Decrypt(data, password []byte) ([]byte, error) {
	gcm, err := GCMFromPassphrase(password)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func EncryptPrivateKey(pk crypto.PrivateKey) ([]byte, error) {
	return nil, nil
}

func DecryptPrivateKey(ciphertext []byte) crypto.PrivateKey {
	return nil
}
