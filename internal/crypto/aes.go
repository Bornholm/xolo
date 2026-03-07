package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"

	"github.com/pkg/errors"
)

// Encrypt encrypts plaintext using AES-GCM with the given 32-byte hex key.
func Encrypt(keyHex, plaintext string) (string, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", errors.Wrap(err, "invalid key hex")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.WithStack(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.WithStack(err)
	}

	nonce, err := RandomBytes(gcm.NonceSize())
	if err != nil {
		return "", errors.WithStack(err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext (hex-encoded) using AES-GCM with the given 32-byte hex key.
func Decrypt(keyHex, ciphertextHex string) (string, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", errors.Wrap(err, "invalid key hex")
	}

	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", errors.Wrap(err, "invalid ciphertext hex")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errors.WithStack(err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.WithStack(err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.Wrap(err, "decryption failed")
	}

	return string(plaintext), nil
}
