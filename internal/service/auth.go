// Package service contains the business rules. It knows nothing about HTTP or SQL.
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Almzaba/AgnosHospitalMiddleware/internal/auth"
	"github.com/Almzaba/AgnosHospitalMiddleware/internal/model"
	"github.com/Almzaba/AgnosHospitalMiddleware/internal/repository"
)

// AuthService manages staff accounts and login.
type AuthService interface {
	CreateStaff(ctx context.Context, username, password, hospitalCode string) (*model.Staff, *model.Hospital, error)
	Login(ctx context.Context, username, password, hospitalCode string) (token string, expiresAt time.Time, err error)
}

type authService struct {
	staff     repository.StaffRepository
	hospitals repository.HospitalRepository
	tokens    *auth.TokenManager
}

// NewAuthService wires the auth service.
func NewAuthService(staff repository.StaffRepository, hospitals repository.HospitalRepository, tokens *auth.TokenManager) AuthService {
	return &authService{staff: staff, hospitals: hospitals, tokens: tokens}
}

func (s *authService) CreateStaff(ctx context.Context, username, password, hospitalCode string) (*model.Staff, *model.Hospital, error) {
	hospital, err := s.hospitals.FindByCode(ctx, strings.TrimSpace(hospitalCode))
	if err != nil {
		return nil, nil, err // model.ErrHospitalNotFound or a DB error
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	staff := &model.Staff{
		Username:     strings.TrimSpace(username),
		PasswordHash: string(hash),
		HospitalID:   hospital.ID,
	}
	if err := s.staff.Create(ctx, staff); err != nil {
		return nil, nil, err
	}
	return staff, hospital, nil
}

// Login deliberately returns the same error for "unknown hospital", "unknown user" and
// "wrong password" so callers cannot probe which usernames exist.
func (s *authService) Login(ctx context.Context, username, password, hospitalCode string) (string, time.Time, error) {
	hospital, err := s.hospitals.FindByCode(ctx, strings.TrimSpace(hospitalCode))
	if err != nil {
		return "", time.Time{}, hideNotFound(err, model.ErrHospitalNotFound)
	}

	staff, err := s.staff.FindByUsername(ctx, strings.TrimSpace(username), hospital.ID)
	if err != nil {
		return "", time.Time{}, hideNotFound(err, model.ErrStaffNotFound)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte(password)); err != nil {
		return "", time.Time{}, model.ErrInvalidCredentials
	}

	return s.tokens.Issue(staff)
}

func hideNotFound(err, notFound error) error {
	if err == notFound {
		return model.ErrInvalidCredentials
	}
	return err
}
