// Package auth issues and verifies the JWTs used by staff.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/your-username/agnos-middleware/internal/model"
)

// Claims is the payload stored inside the token. HospitalID is what scopes patient searches.
type Claims struct {
	StaffID    uint   `json:"staff_id"`
	HospitalID uint   `json:"hospital_id"`
	Username   string `json:"username"`
	jwt.RegisteredClaims
}

// TokenManager signs and verifies HS256 tokens.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokenManager builds a TokenManager.
func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl, now: time.Now}
}

// Issue creates a signed token for the staff member and returns its expiry time.
func (m *TokenManager) Issue(staff *model.Staff) (string, time.Time, error) {
	now := m.now()
	expiresAt := now.Add(m.ttl)
	claims := Claims{
		StaffID:    staff.ID,
		HospitalID: staff.HospitalID,
		Username:   staff.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// Parse verifies the signature and expiry and returns the claims.
func (m *TokenManager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil || !token.Valid {
		return nil, model.ErrInvalidToken
	}
	return claims, nil
}
