package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// QRData represents the structure of data encoded in QR codes
type QRData struct {
	StudentID string `json:"student_id"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
	Version   int    `json:"version"`
}

// QRSecretManager handles QR secret operations
type QRSecretManager struct {
	masterKey string
}

// NewQRSecretManager creates a new QR secret manager
func NewQRSecretManager(masterKey string) *QRSecretManager {
	return &QRSecretManager{
		masterKey: masterKey,
	}
}

// GenerateQRSecret generates a random 32-character hex string for QR codes
func GenerateQRSecret() string {
	bytes := make([]byte, 16) // 16 bytes = 32 hex characters
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateQRData creates QR code data with signature for client-side generation
func (qsm *QRSecretManager) GenerateQRData(studentID, userSecret string) (*QRData, error) {
	timestamp := time.Now().Unix()
	
	// Create signature using HMAC-SHA256
	signature := qsm.createSignature(studentID, userSecret, timestamp)
	
	return &QRData{
		StudentID: studentID,
		Timestamp: timestamp,
		Signature: signature,
		Version:   1,
	}, nil
}

// ValidateQRData validates QR code data and signature
func (qsm *QRSecretManager) ValidateQRData(qrData *QRData, userSecret string, maxAge time.Duration) error {
	// Check timestamp validity
	now := time.Now().Unix()
	if now-qrData.Timestamp > int64(maxAge.Seconds()) {
		return fmt.Errorf("QR code expired")
	}
	
	// Verify signature
	expectedSignature := qsm.createSignature(qrData.StudentID, userSecret, qrData.Timestamp)
	if !hmac.Equal([]byte(qrData.Signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid QR code signature")
	}
	
	return nil
}

// ParseQRData parses QR code JSON data
func ParseQRData(qrJSON string) (*QRData, error) {
	var qrData QRData
	if err := json.Unmarshal([]byte(qrJSON), &qrData); err != nil {
		return nil, fmt.Errorf("invalid QR code format: %v", err)
	}
	
	if qrData.Version != 1 {
		return nil, fmt.Errorf("unsupported QR code version: %d", qrData.Version)
	}
	
	return &qrData, nil
}

// SerializeQRData converts QR data to JSON string for client-side QR generation
func SerializeQRData(qrData *QRData) (string, error) {
	jsonBytes, err := json.Marshal(qrData)
	if err != nil {
		return "", fmt.Errorf("failed to serialize QR data: %v", err)
	}
	return string(jsonBytes), nil
}

// createSignature creates HMAC-SHA256 signature for QR data
func (qsm *QRSecretManager) createSignature(studentID, userSecret string, timestamp int64) string {
	// Combine data for signing
	data := fmt.Sprintf("%s:%s:%d:%s", studentID, userSecret, timestamp, qsm.masterKey)
	
	// Create HMAC
	h := hmac.New(sha256.New, []byte(qsm.masterKey))
	h.Write([]byte(data))
	
	// Return base64 encoded signature
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// RegenerateUserSecret generates a new secret for a user (used for QR refresh)
func RegenerateUserSecret() string {
	return GenerateQRSecret()
}

// GetQRCodeURL generates a URL that can be used by frontend to generate QR code
func GetQRCodeURL(qrData *QRData, baseURL string) string {
	jsonData, _ := SerializeQRData(qrData)
	encoded := base64.URLEncoding.EncodeToString([]byte(jsonData))
	return fmt.Sprintf("%s/qr/%s", baseURL, encoded)
}

// ValidateQECodeWithUserLookup validates QR code and returns user info
type QRValidationResult struct {
	Valid     bool   `json:"valid"`
	StudentID string `json:"student_id"`
	Error     string `json:"error,omitempty"`
	UserID    uint   `json:"user_id,omitempty"`
}

// QRScanContext contains information about QR scan attempt
type QRScanContext struct {
	AdminID    uint      `json:"admin_id"`
	ActivityID uint      `json:"activity_id"`
	ScanTime   time.Time `json:"scan_time"`
	Location   string    `json:"location,omitempty"`
}