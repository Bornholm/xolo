package crypto

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/pkg/errors"
)

func RandomBytes(size int) ([]byte, error) {
	data := make([]byte, size)

	read, err := rand.Read(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if read != size {
		return nil, errors.New("unexpected number of read bytes")
	}

	return data, nil
}

// GenerateSecureToken generates a cryptographically secure token suitable for API authentication
func GenerateSecureToken() (string, error) {
	// Generate 32 bytes (256 bits) of random data for strong security
	bytes, err := RandomBytes(32)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// Encode as base64 URL-safe string (no padding) for easy use in URLs/headers
	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, nil
}
