// Package repository contains the persistence layer. Services depend on the interfaces,
// which keeps business logic testable without a database.
package repository

import (
	"context"

	"github.com/Almzana/AgnosHospitalMiddleware/internal/model"
)

// HospitalRepository reads hospital records.
type HospitalRepository interface {
	FindByCode(ctx context.Context, code string) (*model.Hospital, error)
	FindByID(ctx context.Context, id uint) (*model.Hospital, error)
}

// StaffRepository stores staff accounts.
type StaffRepository interface {
	// Create returns model.ErrStaffExists when (username, hospital) is already taken.
	Create(ctx context.Context, staff *model.Staff) error
	// FindByUsername returns model.ErrStaffNotFound when there is no match.
	FindByUsername(ctx context.Context, username string, hospitalID uint) (*model.Staff, error)
}

// PatientRepository stores patients. Every method is scoped to a hospital.
type PatientRepository interface {
	Search(ctx context.Context, hospitalID uint, filter model.PatientSearchFilter) ([]model.Patient, error)
	// Upsert inserts or updates a patient keyed by (hospital_id, patient_hn).
	Upsert(ctx context.Context, patient *model.Patient) error
}
