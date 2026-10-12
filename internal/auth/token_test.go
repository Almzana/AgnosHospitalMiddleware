package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/your-username/agnos-middleware/internal/model"
)

const secret = "unit-test-secret-0123456789"

var testStaff = &model.Staff{ID: 7, Username: "alice", HospitalID: 3}

func TestIssueAndParse_RoundTrip(t *testing.T) {
	m := NewTokenManager(secret, time.Hour)

	token, expiresAt, err := m.Issue(testStaff)
	if err != nil {
		t.Fatal(err)
	}
	if !expiresAt.After(time.Now()) {
		t.Error("expiry must be in the future")
	}

	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if claims.StaffID != 7 || claims.HospitalID != 3 || claims.Username != "alice" {
		t.Errorf("claims mismatch: %+v", claims)
	}
}

func TestParse_Rejects(t *testing.T) {
	m := NewTokenManager(secret, time.Hour)
	good, _, _ := m.Issue(testStaff)

	// token signed 2 hours ago with a 1 hour lifetime
	past := NewTokenManager(secret, time.Hour)
	past.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	expired, _, _ := past.Issue(testStaff)

	otherSecret, _, _ := NewTokenManager("a-completely-different-secret", time.Hour).Issue(testStaff)

	wrongAlg, err := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{StaffID: 7, HospitalID: 3}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]string{
		"empty":            "",
		"garbage":          "not-a-jwt",
		"expired":          expired,
		"different secret": otherSecret,
		"tampered payload": good[:len(good)-4] + "AAAA",
		"unexpected alg":   wrongAlg,
	}
	for name, tok := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := m.Parse(tok); !errors.Is(err, model.ErrInvalidToken) {
				t.Fatalf("want ErrInvalidToken, got %v", err)
			}
		})
	}
}
