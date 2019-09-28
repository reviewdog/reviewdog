// Package secretbox provides utility wrapper of
// https://godoc.org/golang.org/x/crypto/nacl/secretbox
package secretbox

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

type SecretBox struct {
	key [32]byte
}

func New(key [32]byte) *SecretBox {
	return &SecretBox{key: key}
}

func NewFromHexKey(key string) (*SecretBox, error) {
	secretKeyBytes, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}
	s := &SecretBox{}
	if len(secretKeyBytes) != 32 {
		return nil, fmt.Errorf("key is not 32-byte")
	}
	copy(s.key[:], secretKeyBytes)
	return s, nil
}

func (s *SecretBox) Encrypt(plaintext []byte) ([]byte, error) {
	return Encrypt(plaintext, s.key)
}

func (s *SecretBox) Decrypt(ciphertext []byte) ([]byte, error) {
	return Decrypt(ciphertext, s.key)
}

func Encrypt(plaintext []byte, key [32]byte) ([]byte, error) {
	nonce, err := generateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce")
	}
	return secretbox.Seal(nonce[:], plaintext, &nonce, &key), nil
}

func Decrypt(ciphertext []byte, key [32]byte) ([]byte, error) {
	var nonce [24]byte
	copy(nonce[:], ciphertext[:24])
	decrypted, ok := secretbox.Open(nil, ciphertext[24:], &nonce, &key)
	if !ok {
		return nil, errors.New("failed to decript given message")
	}
	return decrypted, nil
}

func generateNonce() ([24]byte, error) {
	var nonce [24]byte
	_, err := io.ReadFull(rand.Reader, nonce[:])
	return nonce, err
}
