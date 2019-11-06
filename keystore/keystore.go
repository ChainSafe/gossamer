package keystore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"io/ioutil"
	"path/filepath"

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

func EncryptPrivateKey(pk crypto.PrivateKey, password []byte) ([]byte, error) {
	return Encrypt(pk.Encode(), password)
}

func DecryptPrivateKey(data, password []byte) (crypto.PrivateKey, error) {
	pk, err := Decrypt(data, password)
	if err != nil {
		return nil, err
	}

	return crypto.DecodePrivateKey(pk)
}

func EncryptAndWriteToFile(filename string, pk crypto.PrivateKey, password []byte) error {
	data, err := EncryptPrivateKey(pk, password)
	if err != nil {
		return err
	}

	fp, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fp, data, 0644)
}

func ReadFromFileAndDecrypt(filename string, password []byte) (pk crypto.PrivateKey, err error) {
	fp, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}

	return DecryptPrivateKey(data, password)
}
