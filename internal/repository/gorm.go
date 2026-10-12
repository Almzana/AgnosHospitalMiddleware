package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Almzana/AgnosHospitalMiddleware/internal/model"
)

const (
	pgUniqueViolation = "23505"
	maxSearchResults  = 100
)

// ---------- hospitals ----------

type gormHospitalRepo struct{ db *gorm.DB }

// NewHospitalRepository returns a GORM-backed HospitalRepository.
func NewHospitalRepository(db *gorm.DB) HospitalRepository { return &gormHospitalRepo{db: db} }

func (r *gormHospitalRepo) FindByCode(ctx context.Context, code string) (*model.Hospital, error) {
	var h model.Hospital
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&h).Error
	return &h, mapNotFound(err, model.ErrHospitalNotFound)
}

func (r *gormHospitalRepo) FindByID(ctx context.Context, id uint) (*model.Hospital, error) {
	var h model.Hospital
	err := r.db.WithContext(ctx).First(&h, id).Error
	return &h, mapNotFound(err, model.ErrHospitalNotFound)
}

// ---------- staff ----------

type gormStaffRepo struct{ db *gorm.DB }

// NewStaffRepository returns a GORM-backed StaffRepository.
func NewStaffRepository(db *gorm.DB) StaffRepository { return &gormStaffRepo{db: db} }

func (r *gormStaffRepo) Create(ctx context.Context, staff *model.Staff) error {
	err := r.db.WithContext(ctx).Create(staff).Error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return model.ErrStaffExists
	}
	return err
}

func (r *gormStaffRepo) FindByUsername(ctx context.Context, username string, hospitalID uint) (*model.Staff, error) {
	var s model.Staff
	err := r.db.WithContext(ctx).
		Where("username = ? AND hospital_id = ?", username, hospitalID).
		First(&s).Error
	return &s, mapNotFound(err, model.ErrStaffNotFound)
}

// ---------- patients ----------

type gormPatientRepo struct{ db *gorm.DB }

// NewPatientRepository returns a GORM-backed PatientRepository.
func NewPatientRepository(db *gorm.DB) PatientRepository { return &gormPatientRepo{db: db} }

func (r *gormPatientRepo) Search(ctx context.Context, hospitalID uint, f model.PatientSearchFilter) ([]model.Patient, error) {
	// hospital_id is always applied first: this is the tenant boundary.
	q := r.db.WithContext(ctx).Where("hospital_id = ?", hospitalID)

	if f.NationalID != "" {
		q = q.Where("national_id = ?", f.NationalID)
	}
	if f.PassportID != "" {
		q = q.Where("passport_id = ?", f.PassportID)
	}
	// Names are stored in Thai and English, so a name matches either language.
	if f.FirstName != "" {
		q = q.Where("(first_name_th ILIKE ? OR first_name_en ILIKE ?)", like(f.FirstName), like(f.FirstName))
	}
	if f.MiddleName != "" {
		q = q.Where("(middle_name_th ILIKE ? OR middle_name_en ILIKE ?)", like(f.MiddleName), like(f.MiddleName))
	}
	if f.LastName != "" {
		q = q.Where("(last_name_th ILIKE ? OR last_name_en ILIKE ?)", like(f.LastName), like(f.LastName))
	}
	if f.DateOfBirth != nil {
		q = q.Where("date_of_birth = ?", f.DateOfBirth.Format("2006-01-02"))
	}
	if f.PhoneNumber != "" {
		q = q.Where("phone_number = ?", f.PhoneNumber)
	}
	if f.Email != "" {
		q = q.Where("LOWER(email) = LOWER(?)", f.Email)
	}

	var patients []model.Patient
	err := q.Order("id").Limit(maxSearchResults).Find(&patients).Error
	return patients, err
}

func (r *gormPatientRepo) Upsert(ctx context.Context, p *model.Patient) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "hospital_id"}, {Name: "patient_hn"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"first_name_th", "middle_name_th", "last_name_th",
			"first_name_en", "middle_name_en", "last_name_en",
			"date_of_birth", "national_id", "passport_id",
			"phone_number", "email", "gender", "updated_at",
		}),
	}).Create(p).Error
}

// ---------- helpers ----------

func mapNotFound(err, target error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return target
	}
	return err
}

// like builds a "contains" pattern and escapes LIKE wildcards typed by the user.
func like(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}
