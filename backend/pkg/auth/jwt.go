package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JWTClaims struct {
	UserID       uint   `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	FacultyID    *uint  `json:"faculty_id,omitempty"`
	DepartmentID *uint  `json:"department_id,omitempty"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey      string
	expireHours    int
}

func NewJWTService(secretKey string, expireHours int) *JWTService {
	return &JWTService{
		secretKey:   secretKey,
		expireHours: expireHours,
	}
}

func (j *JWTService) GenerateToken(userID uint, email, role string, facultyID, departmentID *uint) (string, error) {
	claims := JWTClaims{
		UserID:       userID,
		Email:        email,
		Role:         role,
		FacultyID:    facultyID,
		DepartmentID: departmentID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(j.expireHours))),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "tru-activity",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

func (j *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (j *JWTService) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Generate new token with same claims but updated expiry
	return j.GenerateToken(claims.UserID, claims.Email, claims.Role, claims.FacultyID, claims.DepartmentID)
}