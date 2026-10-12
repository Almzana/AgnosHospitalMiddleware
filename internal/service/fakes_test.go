package service

import (
	"context"
	"io"
	"log/slog"

	"github.com/Almzana/AgnosHospitalMiddleware/internal/hospitalapi"
	"github.com/Almzana/AgnosHospitalMiddleware/internal/model"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// ---------- fake hospital repo ----------

type fakeHospitalRepo struct {
	byCode map[string]*model.Hospital
	err    error // forced error for every call
}

func newFakeHospitalRepo(hs ...*model.Hospital) *fakeHospitalRepo {
	r := &fakeHospitalRepo{byCode: map[string]*model.Hospital{}}
	for _, h := range hs {
		r.byCode[h.Code] = h
	}
	return r
}

func (r *fakeHospitalRepo) FindByCode(_ context.Context, code string) (*model.Hospital, error) {
	if r.err != nil {
		return nil, r.err
	}
	if h, ok := r.byCode[code]; ok {
		return h, nil
	}
	return nil, model.ErrHospitalNotFound
}

func (r *fakeHospitalRepo) FindByID(_ context.Context, id uint) (*model.Hospital, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, h := range r.byCode {
		if h.ID == id {
			return h, nil
		}
	}
	return nil, model.ErrHospitalNotFound
}

// ---------- fake staff repo ----------

type fakeStaffRepo struct {
	staff    []*model.Staff
	createEr error
	nextID   uint
}

func (r *fakeStaffRepo) Create(_ context.Context, s *model.Staff) error {
	if r.createEr != nil {
		return r.createEr
	}
	for _, e := range r.staff {
		if e.Username == s.Username && e.HospitalID == s.HospitalID {
			return model.ErrStaffExists
		}
	}
	r.nextID++
	s.ID = r.nextID
	r.staff = append(r.staff, s)
	return nil
}

func (r *fakeStaffRepo) FindByUsername(_ context.Context, username string, hospitalID uint) (*model.Staff, error) {
	for _, e := range r.staff {
		if e.Username == username && e.HospitalID == hospitalID {
			return e, nil
		}
	}
	return nil, model.ErrStaffNotFound
}

// ---------- fake patient repo ----------

type fakePatientRepo struct {
	patients    []model.Patient
	searchErr   error
	upsertErr   error
	searchCalls int
	upsertCalls int
	lastHosp    uint
}

// Search mimics the real repository for the criteria used in these tests:
// the hospital boundary plus national_id / passport_id.
func (r *fakePatientRepo) Search(_ context.Context, hospitalID uint, f model.PatientSearchFilter) ([]model.Patient, error) {
	r.searchCalls++
	r.lastHosp = hospitalID
	if r.searchErr != nil {
		return nil, r.searchErr
	}
	var out []model.Patient
	for _, p := range r.patients {
		if p.HospitalID != hospitalID {
			continue
		}
		if f.NationalID != "" && p.NationalID != f.NationalID {
			continue
		}
		if f.PassportID != "" && p.PassportID != f.PassportID {
			continue
		}
		if f.FirstName != "" && p.FirstNameEN != f.FirstName {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *fakePatientRepo) Upsert(_ context.Context, p *model.Patient) error {
	r.upsertCalls++
	if r.upsertErr != nil {
		return r.upsertErr
	}
	for i, e := range r.patients {
		if e.HospitalID == p.HospitalID && e.PatientHN == p.PatientHN {
			r.patients[i] = *p
			return nil
		}
	}
	r.patients = append(r.patients, *p)
	return nil
}

// ---------- fake HIS client ----------

type fakeHIS struct {
	patient *hospitalapi.RemotePatient
	err     error
	calls   int
	baseURL string
	id      string
}

func (h *fakeHIS) FetchPatient(_ context.Context, baseURL, id string) (*hospitalapi.RemotePatient, error) {
	h.calls++
	h.baseURL, h.id = baseURL, id
	if h.err != nil {
		return nil, h.err
	}
	return h.patient, nil
}
