package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/your-username/agnos-middleware/internal/hospitalapi"
	"github.com/your-username/agnos-middleware/internal/model"
	"github.com/your-username/agnos-middleware/internal/repository"
)

// PatientService searches patients on behalf of a staff member.
type PatientService interface {
	// Search only ever returns patients of hospitalID (taken from the staff member's token).
	Search(ctx context.Context, hospitalID uint, filter model.PatientSearchFilter) ([]model.Patient, error)
}

type patientService struct {
	patients  repository.PatientRepository
	hospitals repository.HospitalRepository
	his       hospitalapi.Client
	logger    *slog.Logger
}

// NewPatientService wires the patient service.
func NewPatientService(
	patients repository.PatientRepository,
	hospitals repository.HospitalRepository,
	his hospitalapi.Client,
	logger *slog.Logger,
) PatientService {
	return &patientService{patients: patients, hospitals: hospitals, his: his, logger: logger}
}

// Search looks in the local database first. When the caller searched by national_id or
// passport_id and nothing was found locally, the hospital's HIS is queried and the result is
// cached in the local database so later (multi-criteria) searches can find it.
func (s *patientService) Search(ctx context.Context, hospitalID uint, filter model.PatientSearchFilter) ([]model.Patient, error) {
	filter.Normalize()

	found, err := s.patients.Search(ctx, hospitalID, filter)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	if len(found) > 0 || filter.Identifier() == "" {
		return found, nil
	}

	remote, err := s.fetchFromHIS(ctx, hospitalID, filter.Identifier())
	if err != nil {
		// HIS being down must not break searching; log it and answer with what we have.
		if !errors.Is(err, model.ErrPatientNotFound) {
			s.logger.Warn("HIS lookup failed", "hospital_id", hospitalID, "error", err)
		}
		return found, nil
	}
	if remote == nil {
		return found, nil
	}

	// Cache, then re-run the full filter: the HIS only matches on the id, so the other
	// criteria the caller supplied still have to hold.
	if err := s.patients.Upsert(ctx, remote); err != nil {
		return nil, fmt.Errorf("cache patient: %w", err)
	}
	found, err = s.patients.Search(ctx, hospitalID, filter)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	return found, nil
}

// fetchFromHIS returns (nil, nil) when the hospital has no HIS configured.
func (s *patientService) fetchFromHIS(ctx context.Context, hospitalID uint, id string) (*model.Patient, error) {
	hospital, err := s.hospitals.FindByID(ctx, hospitalID)
	if err != nil {
		return nil, err
	}
	if hospital.APIBaseURL == "" {
		return nil, nil
	}

	rp, err := s.his.FetchPatient(ctx, hospital.APIBaseURL, id)
	if err != nil {
		return nil, err
	}
	return toPatient(hospitalID, rp)
}

// toPatient maps the HIS payload onto our model and validates the essentials.
func toPatient(hospitalID uint, rp *hospitalapi.RemotePatient) (*model.Patient, error) {
	if rp.PatientHN == "" {
		return nil, errors.New("HIS response is missing patient_hn")
	}
	if rp.Gender != "" && rp.Gender != "M" && rp.Gender != "F" {
		return nil, fmt.Errorf("HIS response has invalid gender %q", rp.Gender)
	}

	dob, err := parseDate(rp.DateOfBirth)
	if err != nil {
		return nil, fmt.Errorf("HIS response has invalid date_of_birth: %w", err)
	}

	return &model.Patient{
		HospitalID:   hospitalID,
		FirstNameTH:  rp.FirstNameTH,
		MiddleNameTH: rp.MiddleNameTH,
		LastNameTH:   rp.LastNameTH,
		FirstNameEN:  rp.FirstNameEN,
		MiddleNameEN: rp.MiddleNameEN,
		LastNameEN:   rp.LastNameEN,
		DateOfBirth:  dob,
		PatientHN:    rp.PatientHN,
		NationalID:   rp.NationalID,
		PassportID:   rp.PassportID,
		PhoneNumber:  rp.PhoneNumber,
		Email:        rp.Email,
		Gender:       rp.Gender,
	}, nil
}

func parseDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return &d, nil
		}
	}
	return nil, fmt.Errorf("unsupported format %q", s)
}
