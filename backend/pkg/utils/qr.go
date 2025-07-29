package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateQRSecret generates a random 32-character hex string for QR codes
func GenerateQRSecret() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 32 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}