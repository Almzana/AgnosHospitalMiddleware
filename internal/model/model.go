// Package model holds the domain types shared by every layer.
package model

import (
	"strings"
	"time"
)

// Hospital is a tenant of the middleware. Every staff member and patient belongs to exactly one.
type Hospital struct {
	ID         uint   `gorm:"primaryKey"`
	Code       string `gorm:"uniqueIndex;size:50"`
	Name       string
	APIBaseURL string `gorm:"column:api_base_url"`
	CreatedAt  time.Time
}

// Staff is a hospital employee who can log in and search patients of their own hospital.
type Staff struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"size:100"`
	PasswordHash string
	HospitalID   uint
	CreatedAt    time.Time
}

// TableName keeps the table name aligned with the SQL migration.
func (Staff) TableName() string { return "staff" }

// Patient mirrors the HIS payload plus the owning hospital.
type Patient struct {
	ID           uint `gorm:"primaryKey"`
	HospitalID   uint
	FirstNameTH  string `gorm:"column:first_name_th"`
	MiddleNameTH string `gorm:"column:middle_name_th"`
	LastNameTH   string `gorm:"column:last_name_th"`
	FirstNameEN  string `gorm:"column:first_name_en"`
	MiddleNameEN string `gorm:"column:middle_name_en"`
	LastNameEN   string `gorm:"column:last_name_en"`
	DateOfBirth  *time.Time
	PatientHN    string `gorm:"column:patient_hn"`
	NationalID   string `gorm:"column:national_id"`
	PassportID   string `gorm:"column:passport_id"`
	PhoneNumber  string
	Email        string
	Gender       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// PatientSearchFilter holds the optional criteria of /patient/search.
// An empty field means "do not filter on this".
type PatientSearchFilter struct {
	NationalID  string
	PassportID  string
	FirstName   string
	MiddleName  string
	LastName    string
	DateOfBirth *time.Time
	PhoneNumber string
	Email       string
}

// Normalize trims whitespace from every text criterion.
func (f *PatientSearchFilter) Normalize() {
	f.NationalID = strings.TrimSpace(f.NationalID)
	f.PassportID = strings.TrimSpace(f.PassportID)
	f.FirstName = strings.TrimSpace(f.FirstName)
	f.MiddleName = strings.TrimSpace(f.MiddleName)
	f.LastName = strings.TrimSpace(f.LastName)
	f.PhoneNumber = strings.TrimSpace(f.PhoneNumber)
	f.Email = strings.TrimSpace(f.Email)
}

// Identifier returns the national or passport id when one was supplied.
// These are the only criteria the HIS API can look up by.
func (f PatientSearchFilter) Identifier() string {
	if f.NationalID != "" {
		return f.NationalID
	}
	return f.PassportID
}
