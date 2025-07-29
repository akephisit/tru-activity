package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateQRSecret generates a random 32-character hex string for QR codes
func GenerateQRSecret() string {
	bytes := make([]byte, 16) // 16 bytes = 32 hex characters
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}