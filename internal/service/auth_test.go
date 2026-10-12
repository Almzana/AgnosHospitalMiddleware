package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/your-username/agnos-middleware/internal/auth"
	"github.com/your-username/agnos-middleware/internal/model"
)

var (
	hospA = &model.Hospital{ID: 1, Code: "hospital-a", Name: "Hospital A"}
	hospB = &model.Hospital{ID: 2, Code: "hospital-b", Name: "Hospital B"}
)

func newAuthSvc(t *testing.T) (AuthService, *fakeStaffRepo, *auth.TokenManager) {
	t.Helper()
	staff := &fakeStaffRepo{}
	tokens := auth.NewTokenManager("unit-test-secret-0123456789", time.Hour)
	return NewAuthService(staff, newFakeHospitalRepo(hospA, hospB), tokens), staff, tokens
}

func TestCreateStaff_Success(t *testing.T) {
	svc, repo, _ := newAuthSvc(t)

	staff, hospital, err := svc.CreateStaff(context.Background(), "alice", "s3cret-pass", "hospital-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if staff.HospitalID != hospA.ID || hospital.Code != "hospital-a" {
		t.Errorf("staff attached to wrong hospital: %+v / %+v", staff, hospital)
	}
	if staff.PasswordHash == "" || staff.PasswordHash == "s3cret-pass" {
		t.Error("password must be stored hashed, never in plain text")
	}
	if len(repo.staff) != 1 {
		t.Errorf("expected 1 stored staff, got %d", len(repo.staff))
	}
}

func TestCreateStaff_UnknownHospital(t *testing.T) {
	svc, repo, _ := newAuthSvc(t)

	_, _, err := svc.CreateStaff(context.Background(), "alice", "s3cret-pass", "nope")
	if !errors.Is(err, model.ErrHospitalNotFound) {
		t.Fatalf("want ErrHospitalNotFound, got %v", err)
	}
	if len(repo.staff) != 0 {
		t.Error("nothing should be stored when the hospital is unknown")
	}
}

func TestCreateStaff_DuplicateInSameHospital(t *testing.T) {
	svc, _, _ := newAuthSvc(t)
	ctx := context.Background()

	if _, _, err := svc.CreateStaff(ctx, "alice", "s3cret-pass", "hospital-a"); err != nil {
		t.Fatal(err)
	}
	_, _, err := svc.CreateStaff(ctx, "alice", "another-pass", "hospital-a")
	if !errors.Is(err, model.ErrStaffExists) {
		t.Fatalf("want ErrStaffExists, got %v", err)
	}
}

func TestCreateStaff_SameUsernameInOtherHospitalAllowed(t *testing.T) {
	svc, _, _ := newAuthSvc(t)
	ctx := context.Background()

	if _, _, err := svc.CreateStaff(ctx, "alice", "s3cret-pass", "hospital-a"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.CreateStaff(ctx, "alice", "s3cret-pass", "hospital-b"); err != nil {
		t.Fatalf("same username in a different hospital must be allowed: %v", err)
	}
}

func TestCreateStaff_RepositoryError(t *testing.T) {
	svc, repo, _ := newAuthSvc(t)
	repo.createEr = errors.New("db down")

	_, _, err := svc.CreateStaff(context.Background(), "alice", "s3cret-pass", "hospital-a")
	if err == nil || errors.Is(err, model.ErrStaffExists) {
		t.Fatalf("want the underlying repository error, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc, _, tokens := newAuthSvc(t)
	ctx := context.Background()
	if _, _, err := svc.CreateStaff(ctx, "alice", "s3cret-pass", "hospital-b"); err != nil {
		t.Fatal(err)
	}

	token, expiresAt, err := svc.Login(ctx, "alice", "s3cret-pass", "hospital-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Error("expiry should be in the future")
	}
	claims, err := tokens.Parse(token)
	if err != nil {
		t.Fatalf("issued token must be valid: %v", err)
	}
	if claims.HospitalID != hospB.ID || claims.Username != "alice" {
		t.Errorf("token carries wrong identity: %+v", claims)
	}
}

func TestLogin_Failures_AreIndistinguishable(t *testing.T) {
	svc, _, _ := newAuthSvc(t)
	ctx := context.Background()
	if _, _, err := svc.CreateStaff(ctx, "alice", "s3cret-pass", "hospital-a"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, user, pass, hospital string
	}{
		{"wrong password", "alice", "wrong-pass", "hospital-a"},
		{"unknown user", "bob", "s3cret-pass", "hospital-a"},
		{"unknown hospital", "alice", "s3cret-pass", "nope"},
		{"user exists only in another hospital", "alice", "s3cret-pass", "hospital-b"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, _, err := svc.Login(ctx, tc.user, tc.pass, tc.hospital)
			if !errors.Is(err, model.ErrInvalidCredentials) {
				t.Fatalf("want ErrInvalidCredentials, got %v", err)
			}
			if token != "" {
				t.Error("no token may be issued on failure")
			}
		})
	}
}

func TestLogin_InfrastructureErrorIsNotHidden(t *testing.T) {
	svc, _, _ := newAuthSvc(t)
	hosp := newFakeHospitalRepo(hospA)
	hosp.err = errors.New("db down")
	svc = NewAuthService(&fakeStaffRepo{}, hosp, auth.NewTokenManager("unit-test-secret-0123456789", time.Hour))

	_, _, err := svc.Login(context.Background(), "alice", "s3cret-pass", "hospital-a")
	if err == nil || errors.Is(err, model.ErrInvalidCredentials) {
		t.Fatalf("a DB failure must not look like bad credentials, got %v", err)
	}
}
