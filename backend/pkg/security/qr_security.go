package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/scrypt"
	"context"
)

const (
	// QR Security Constants
	QRSecretLength     = 32
	QRExpiryDuration   = 15 * time.Minute
	MaxQRScanAttempts  = 5
	QRSignatureVersion = 2
	
	// Redis Keys
	QRSecretKey        = "qr_secret:"
	QRUsageKey         = "qr_usage:"
	QRScanAttemptKey   = "qr_scan_attempt:"
	QRBlacklistKey     = "qr_blacklist:"
	
	// Security Parameters
	ScryptN = 32768
	ScryptR = 8
	ScryptP = 1
)

type QRSecurityManager struct {
	redisClient   *redis.Client
	masterSecret  []byte
	signatureKey  []byte
}

type QRData struct {
	StudentID   string `json:"student_id"`
	Timestamp   int64  `json:"timestamp"`
	Nonce       string `json:"nonce"`
	Version     int    `json:"version"`
	Signature   string `json:"signature"`
	SecretHash  string `json:"secret_hash"`
}

type QRValidationResult struct {
	Valid         bool   `json:"valid"`
	StudentID     string `json:"student_id,omitempty"`
	Message       string `json:"message"`
	SecurityLevel string `json:"security_level"`
	Timestamp     int64  `json:"timestamp"`
}

type QRScanAttempt struct {
	UserID        string    `json:"user_id"`
	QRData        string    `json:"qr_data"`
	Timestamp     time.Time `json:"timestamp"`
	Success       bool      `json:"success"`
	ErrorReason   string    `json:"error_reason,omitempty"`
	IPAddress     string    `json:"ip_address"`
	UserAgent     string    `json:"user_agent"`
	ActivityID    string    `json:"activity_id"`
	ScannerID     string    `json:"scanner_id"`
}

func NewQRSecurityManager(redisClient *redis.Client, masterSecret []byte) *QRSecurityManager {
	// Derive signature key from master secret
	signatureKey := sha256.Sum256(append(masterSecret, []byte("qr_signature")...))
	
	return &QRSecurityManager{
		redisClient:  redisClient,
		masterSecret: masterSecret,
		signatureKey: signatureKey[:],
	}
}

// Generate secure QR data for a student
func (qsm *QRSecurityManager) GenerateQRData(ctx context.Context, studentID string) (*QRData, error) {
	// Get or generate user's QR secret
	secret, err := qsm.getUserQRSecret(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get QR secret: %v", err)
	}
	
	// Generate nonce for this QR code
	nonce, err := qsm.generateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}
	
	timestamp := time.Now().Unix()
	
	// Create secret hash using scrypt for additional security
	secretHash, err := qsm.hashSecret(secret, studentID, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to hash secret: %v", err)
	}
	
	qrData := &QRData{
		StudentID:  studentID,
		Timestamp:  timestamp,
		Nonce:      nonce,
		Version:    QRSignatureVersion,
		SecretHash: secretHash,
	}
	
	// Generate signature
	signature, err := qsm.signQRData(qrData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign QR data: %v", err)
	}
	
	qrData.Signature = signature
	
	// Track QR usage
	if err := qsm.trackQRGeneration(ctx, studentID, qrData); err != nil {
		// Log error but don't fail - this is for monitoring
		fmt.Printf("Warning: failed to track QR generation: %v\n", err)
	}
	
	return qrData, nil
}

// Validate QR data with comprehensive security checks
func (qsm *QRSecurityManager) ValidateQRData(ctx context.Context, qrDataStr string, scannerID string, activityID string, clientIP string, userAgent string) (*QRValidationResult, error) {
	result := &QRValidationResult{
		Valid:         false,
		SecurityLevel: "high",
		Timestamp:     time.Now().Unix(),
	}
	
	// Log scan attempt
	attempt := &QRScanAttempt{
		QRData:     qrDataStr,
		Timestamp:  time.Now(),
		IPAddress:  clientIP,
		UserAgent:  userAgent,
		ActivityID: activityID,
		ScannerID:  scannerID,
	}
	
	defer func() {
		// Always log the scan attempt
		qsm.logScanAttempt(ctx, attempt)
	}()
	
	// 1. Parse QR data
	var qrData QRData
	if err := json.Unmarshal([]byte(qrDataStr), &qrData); err != nil {
		result.Message = "Invalid QR data format"
		attempt.ErrorReason = "parse_error"
		return result, nil
	}
	
	attempt.UserID = qrData.StudentID
	
	// 2. Check if QR is blacklisted
	if blacklisted, err := qsm.isQRBlacklisted(ctx, qrData.Signature); err != nil {
		result.Message = "QR validation service error"
		attempt.ErrorReason = "service_error"
		return result, fmt.Errorf("blacklist check failed: %v", err)
	} else if blacklisted {
		result.Message = "QR code has been revoked"
		attempt.ErrorReason = "blacklisted"
		return result, nil
	}
	
	// 3. Version check
	if qrData.Version != QRSignatureVersion {
		result.Message = "QR code version not supported"
		attempt.ErrorReason = "version_mismatch"
		return result, nil
	}
	
	// 4. Timestamp validation (prevent expired and future QR codes)
	now := time.Now().Unix()
	qrTime := qrData.Timestamp
	
	if now-qrTime > int64(QRExpiryDuration.Seconds()) {
		result.Message = "QR code has expired"
		attempt.ErrorReason = "expired"
		return result, nil
	}
	
	if qrTime > now+60 { // Allow 1 minute clock skew
		result.Message = "QR code timestamp is invalid"
		attempt.ErrorReason = "future_timestamp"
		return result, nil
	}
	
	// 5. Replay attack prevention
	if used, err := qsm.isQRUsed(ctx, qrData.Signature); err != nil {
		result.Message = "QR validation service error"
		attempt.ErrorReason = "service_error"
		return result, fmt.Errorf("replay check failed: %v", err)
	} else if used {
		result.Message = "QR code has already been used"
		attempt.ErrorReason = "replay_attack"
		return result, nil
	}
	
	// 6. Rate limiting check
	if exceeded, err := qsm.checkScanRateLimit(ctx, qrData.StudentID); err != nil {
		result.Message = "QR validation service error"
		attempt.ErrorReason = "service_error"
		return result, fmt.Errorf("rate limit check failed: %v", err)
	} else if exceeded {
		result.Message = "Too many scan attempts"
		attempt.ErrorReason = "rate_limited"
		return result, nil
	}
	
	// 7. Signature validation
	if valid, err := qsm.verifyQRSignature(&qrData); err != nil {
		result.Message = "QR signature validation error"
		attempt.ErrorReason = "signature_error"
		return result, fmt.Errorf("signature validation failed: %v", err)
	} else if !valid {
		result.Message = "QR code signature is invalid"
		attempt.ErrorReason = "invalid_signature"
		return result, nil
	}
	
	// 8. Secret validation
	if valid, err := qsm.validateSecret(ctx, &qrData); err != nil {
		result.Message = "QR secret validation error"
		attempt.ErrorReason = "secret_error"
		return result, fmt.Errorf("secret validation failed: %v", err)
	} else if !valid {
		result.Message = "QR secret is invalid"
		attempt.ErrorReason = "invalid_secret"
		return result, nil
	}
	
	// 9. Mark QR as used to prevent replay
	if err := qsm.markQRUsed(ctx, qrData.Signature, qrData.StudentID); err != nil {
		// This is critical - if we can't mark it as used, reject the scan
		result.Message = "QR processing error"
		attempt.ErrorReason = "mark_used_error"
		return result, fmt.Errorf("failed to mark QR as used: %v", err)
	}
	
	// Success!
	result.Valid = true
	result.StudentID = qrData.StudentID
	result.Message = "QR code is valid"
	attempt.Success = true
	
	return result, nil
}

// Get or generate QR secret for a user
func (qsm *QRSecurityManager) getUserQRSecret(ctx context.Context, studentID string) ([]byte, error) {
	key := QRSecretKey + studentID
	
	// Try to get existing secret
	secretStr, err := qsm.redisClient.Get(ctx, key).Result()
	if err == nil {
		return base64.StdEncoding.DecodeString(secretStr)
	}
	
	// Generate new secret
	secret := make([]byte, QRSecretLength)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("failed to generate random secret: %v", err)
	}
	
	// Store secret (expires in 30 days, will be regenerated)
	secretB64 := base64.StdEncoding.EncodeToString(secret)
	if err := qsm.redisClient.Set(ctx, key, secretB64, 30*24*time.Hour).Err(); err != nil {
		return nil, fmt.Errorf("failed to store secret: %v", err)
	}
	
	return secret, nil
}

// Generate cryptographically secure nonce
func (qsm *QRSecurityManager) generateNonce() (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce), nil
}

// Hash secret using scrypt for additional security
func (qsm *QRSecurityManager) hashSecret(secret []byte, studentID string, timestamp int64) (string, error) {
	// Use studentID and timestamp as salt
	salt := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", studentID, timestamp)))
	
	hash, err := scrypt.Key(secret, salt[:], ScryptN, ScryptR, ScryptP, 32)
	if err != nil {
		return "", err
	}
	
	return hex.EncodeToString(hash), nil
}

// Sign QR data using HMAC-SHA256
func (qsm *QRSecurityManager) signQRData(qrData *QRData) (string, error) {
	// Create signing payload (without signature field)
	payload := fmt.Sprintf("%s:%d:%s:%d:%s", 
		qrData.StudentID, 
		qrData.Timestamp, 
		qrData.Nonce, 
		qrData.Version,
		qrData.SecretHash,
	)
	
	mac := hmac.New(sha256.New, qsm.signatureKey)
	mac.Write([]byte(payload))
	signature := mac.Sum(nil)
	
	return hex.EncodeToString(signature), nil
}

// Verify QR signature
func (qsm *QRSecurityManager) verifyQRSignature(qrData *QRData) (bool, error) {
	// Recreate the signature
	expectedSignature, err := qsm.signQRData(qrData)
	if err != nil {
		return false, err
	}
	
	// Constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(qrData.Signature), []byte(expectedSignature)), nil
}

// Validate the secret hash
func (qsm *QRSecurityManager) validateSecret(ctx context.Context, qrData *QRData) (bool, error) {
	// Get user's current secret
	secret, err := qsm.getUserQRSecret(ctx, qrData.StudentID)
	if err != nil {
		return false, err
	}
	
	// Recreate the secret hash
	expectedHash, err := qsm.hashSecret(secret, qrData.StudentID, qrData.Timestamp)
	if err != nil {
		return false, err
	}
	
	return qrData.SecretHash == expectedHash, nil
}

// Check if QR has been used (prevent replay attacks)
func (qsm *QRSecurityManager) isQRUsed(ctx context.Context, signature string) (bool, error) {
	key := QRUsageKey + signature
	exists, err := qsm.redisClient.Exists(ctx, key).Result()
	return exists > 0, err
}

// Mark QR as used
func (qsm *QRSecurityManager) markQRUsed(ctx context.Context, signature string, studentID string) error {
	key := QRUsageKey + signature
	
	// Store with expiry (longer than QR expiry to prevent replay)
	usage := map[string]interface{}{
		"student_id": studentID,
		"used_at":    time.Now().Unix(),
	}
	
	pipe := qsm.redisClient.Pipeline()
	pipe.HMSet(ctx, key, usage)
	pipe.Expire(ctx, key, 2*QRExpiryDuration) // Double the QR expiry
	_, err := pipe.Exec(ctx)
	
	return err
}

// Check scan rate limiting
func (qsm *QRSecurityManager) checkScanRateLimit(ctx context.Context, studentID string) (bool, error) {
	key := QRScanAttemptKey + studentID
	
	// Allow MaxQRScanAttempts per minute
	count, err := qsm.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	
	if count == 1 {
		// Set expiry on first increment
		qsm.redisClient.Expire(ctx, key, time.Minute)
	}
	
	return count > MaxQRScanAttempts, nil
}

// Check if QR is blacklisted
func (qsm *QRSecurityManager) isQRBlacklisted(ctx context.Context, signature string) (bool, error) {
	key := QRBlacklistKey + signature
	exists, err := qsm.redisClient.Exists(ctx, key).Result()
	return exists > 0, err
}

// Blacklist a QR code (e.g., if user reports it as compromised)
func (qsm *QRSecurityManager) BlacklistQR(ctx context.Context, signature string, reason string) error {
	key := QRBlacklistKey + signature
	
	blacklistEntry := map[string]interface{}{
		"blacklisted_at": time.Now().Unix(),
		"reason":        reason,
	}
	
	// Blacklist for 24 hours
	pipe := qsm.redisClient.Pipeline()
	pipe.HMSet(ctx, key, blacklistEntry)
	pipe.Expire(ctx, key, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	
	return err
}

// Regenerate QR secret for a user (e.g., if compromised)
func (qsm *QRSecurityManager) RegenerateUserSecret(ctx context.Context, studentID string) error {
	key := QRSecretKey + studentID
	
	// Delete existing secret
	if err := qsm.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete old secret: %v", err)
	}
	
	// Generate new secret will happen automatically on next QR generation
	return nil
}

// Track QR generation for monitoring
func (qsm *QRSecurityManager) trackQRGeneration(ctx context.Context, studentID string, qrData *QRData) error {
	event := map[string]interface{}{
		"student_id": studentID,
		"timestamp":  qrData.Timestamp,
		"nonce":      qrData.Nonce,
		"version":    qrData.Version,
		"generated_at": time.Now().Unix(),
	}
	
	// Store generation event
	key := fmt.Sprintf("qr_generation:%s:%d", studentID, qrData.Timestamp)
	pipe := qsm.redisClient.Pipeline()
	pipe.HMSet(ctx, key, event)
	pipe.Expire(ctx, key, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	
	return err
}

// Log scan attempt for security monitoring
func (qsm *QRSecurityManager) logScanAttempt(ctx context.Context, attempt *QRScanAttempt) {
	// Convert to map for Redis storage
	attemptMap := map[string]interface{}{
		"user_id":      attempt.UserID,
		"timestamp":    attempt.Timestamp.Unix(),
		"success":      attempt.Success,
		"ip_address":   attempt.IPAddress,
		"user_agent":   attempt.UserAgent,
		"activity_id":  attempt.ActivityID,
		"scanner_id":   attempt.ScannerID,
	}
	
	if attempt.ErrorReason != "" {
		attemptMap["error_reason"] = attempt.ErrorReason
	}
	
	// Store attempt log
	key := fmt.Sprintf("qr_scan_log:%s:%d", attempt.ScannerID, attempt.Timestamp.Unix())
	pipe := qsm.redisClient.Pipeline()
	pipe.HMSet(ctx, key, attemptMap)
	pipe.Expire(ctx, key, 7*24*time.Hour) // Keep logs for 7 days
	_, err := pipe.Exec(ctx)
	
	if err != nil {
		fmt.Printf("Warning: failed to log scan attempt: %v\n", err)
	}
	
	// Update security metrics
	qsm.updateSecurityMetrics(ctx, attempt)
}

// Update security metrics for monitoring
func (qsm *QRSecurityManager) updateSecurityMetrics(ctx context.Context, attempt *QRScanAttempt) {
	// Daily metrics
	today := time.Now().Format("2006-01-02")
	
	pipe := qsm.redisClient.Pipeline()
	
	// Total scans
	pipe.Incr(ctx, fmt.Sprintf("metrics:qr_scans:%s", today))
	
	// Success/failure counts
	if attempt.Success {
		pipe.Incr(ctx, fmt.Sprintf("metrics:qr_success:%s", today))
	} else {
		pipe.Incr(ctx, fmt.Sprintf("metrics:qr_failure:%s", today))
		
		// Track failure reasons
		if attempt.ErrorReason != "" {
			pipe.Incr(ctx, fmt.Sprintf("metrics:qr_failure:%s:%s", today, attempt.ErrorReason))
		}
	}
	
	// Per-user metrics
	if attempt.UserID != "" {
		pipe.Incr(ctx, fmt.Sprintf("metrics:user_scans:%s:%s", attempt.UserID, today))
	}
	
	// Per-scanner metrics
	pipe.Incr(ctx, fmt.Sprintf("metrics:scanner_scans:%s:%s", attempt.ScannerID, today))
	
	// Set expiry for all metrics (keep for 30 days)
	// This would need to be done for each key individually in a real implementation
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to update security metrics: %v\n", err)
	}
}

// Get security metrics for monitoring dashboard
func (qsm *QRSecurityManager) GetSecurityMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})
	
	// Get metrics for the last N days
	for i := 0; i < days; i++ {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		
		// Get daily metrics
		totalScans, _ := qsm.redisClient.Get(ctx, fmt.Sprintf("metrics:qr_scans:%s", date)).Int64()
		successScans, _ := qsm.redisClient.Get(ctx, fmt.Sprintf("metrics:qr_success:%s", date)).Int64()
		failureScans, _ := qsm.redisClient.Get(ctx, fmt.Sprintf("metrics:qr_failure:%s", date)).Int64()
		
		metrics[date] = map[string]interface{}{
			"total_scans":   totalScans,
			"success_scans": successScans,
			"failure_scans": failureScans,
		}
	}
	
	return metrics, nil
}

// Generate QR string for client-side QR code generation
func (qsm *QRSecurityManager) GenerateQRString(ctx context.Context, studentID string) (string, error) {
	qrData, err := qsm.GenerateQRData(ctx, studentID)
	if err != nil {
		return "", err
	}
	
	// Convert to JSON string
	jsonData, err := json.Marshal(qrData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal QR data: %v", err)
	}
	
	// Base64 encode for QR code
	return base64.StdEncoding.EncodeToString(jsonData), nil
}