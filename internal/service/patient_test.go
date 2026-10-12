package service

import (
	"context"
	"errors"
	"testing"

	"github.com/your-username/agnos-middleware/internal/hospitalapi"
	"github.com/your-username/agnos-middleware/internal/model"
)

const hisURL = "https://hospital-a.api.co.th"

func hospitalWithHIS() *model.Hospital {
	return &model.Hospital{ID: 1, Code: "hospital-a", APIBaseURL: hisURL}
}

func newPatientSvc(repo *fakePatientRepo, his *fakeHIS, hospitals ...*model.Hospital) PatientService {
	return NewPatientService(repo, newFakeHospitalRepo(hospitals...), his, discardLogger())
}

func remote(hn, nationalID string) *hospitalapi.RemotePatient {
	return &hospitalapi.RemotePatient{
		FirstNameEN: "Somchai", LastNameEN: "Jaidee",
		FirstNameTH: "สมชาย", LastNameTH: "ใจดี",
		DateOfBirth: "1990-05-17", PatientHN: hn, NationalID: nationalID,
		PhoneNumber: "0812345678", Email: "somchai@example.com", Gender: "M",
	}
}

func TestSearch_LocalHit_DoesNotCallHIS(t *testing.T) {
	repo := &fakePatientRepo{patients: []model.Patient{{HospitalID: 1, PatientHN: "HN1", NationalID: "111"}}}
	his := &fakeHIS{}
	svc := newPatientSvc(repo, his, hospitalWithHIS())

	got, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{NationalID: "111"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].PatientHN != "HN1" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if his.calls != 0 {
		t.Errorf("HIS must not be called when the patient is already known, calls=%d", his.calls)
	}
}

func TestSearch_OnlyReturnsOwnHospital(t *testing.T) {
	repo := &fakePatientRepo{patients: []model.Patient{
		{HospitalID: 1, PatientHN: "A1", NationalID: "111"},
		{HospitalID: 2, PatientHN: "B1", NationalID: "111"}, // same national id, other hospital
	}}
	svc := newPatientSvc(repo, &fakeHIS{}, hospitalWithHIS())

	got, err := svc.Search(context.Background(), 2, model.PatientSearchFilter{NationalID: "111"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].HospitalID != 2 {
		t.Fatalf("staff of hospital 2 must only see hospital 2 patients, got %+v", got)
	}
	if repo.lastHosp != 2 {
		t.Errorf("repository was queried with hospital %d, want 2", repo.lastHosp)
	}
}

func TestSearch_NoCriteria_ReturnsAllOfHospitalWithoutHIS(t *testing.T) {
	repo := &fakePatientRepo{patients: []model.Patient{
		{HospitalID: 1, PatientHN: "A1"}, {HospitalID: 1, PatientHN: "A2"}, {HospitalID: 2, PatientHN: "B1"},
	}}
	his := &fakeHIS{}
	svc := newPatientSvc(repo, his, hospitalWithHIS())

	got, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 patients of hospital 1, got %d", len(got))
	}
	if his.calls != 0 {
		t.Error("HIS must not be called without an id")
	}
}

func TestSearch_FallsBackToHISAndCaches(t *testing.T) {
	repo := &fakePatientRepo{}
	his := &fakeHIS{patient: remote("HN9", "1234567890123")}
	svc := newPatientSvc(repo, his, hospitalWithHIS())

	got, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{NationalID: " 1234567890123 "})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].PatientHN != "HN9" {
		t.Fatalf("expected the HIS patient back, got %+v", got)
	}
	if his.calls != 1 || his.baseURL != hisURL || his.id != "1234567890123" {
		t.Errorf("HIS called wrongly: calls=%d url=%q id=%q", his.calls, his.baseURL, his.id)
	}
	if repo.upsertCalls != 1 || repo.patients[0].HospitalID != 1 {
		t.Errorf("patient must be cached under the staff's hospital: %+v", repo.patients)
	}
	if got[0].DateOfBirth == nil || got[0].DateOfBirth.Format("2006-01-02") != "1990-05-17" {
		t.Errorf("date of birth not mapped: %v", got[0].DateOfBirth)
	}
}

func TestSearch_FallbackUsesPassportID(t *testing.T) {
	repo := &fakePatientRepo{}
	rp := remote("HN7", "")
	rp.PassportID = "AA1234567"
	his := &fakeHIS{patient: rp}
	svc := newPatientSvc(repo, his, hospitalWithHIS())

	got, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{PassportID: "AA1234567"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || his.id != "AA1234567" {
		t.Fatalf("passport lookup failed: got=%+v id=%q", got, his.id)
	}
}

func TestSearch_OtherCriteriaStillApplyAfterHISLookup(t *testing.T) {
	repo := &fakePatientRepo{}
	his := &fakeHIS{patient: remote("HN9", "123")} // first name is Somchai
	svc := newPatientSvc(repo, his, hospitalWithHIS())

	got, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{NationalID: "123", FirstName: "Somsri"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("patient does not match first_name, must not be returned: %+v", got)
	}
}

func TestSearch_HISNotFound_ReturnsEmpty(t *testing.T) {
	repo := &fakePatientRepo{}
	his := &fakeHIS{err: model.ErrPatientNotFound}
	svc := newPatientSvc(repo, his, hospitalWithHIS())

	got, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{NationalID: "999"})
	if err != nil {
		t.Fatalf("a missing patient is not an error: %v", err)
	}
	if len(got) != 0 || repo.upsertCalls != 0 {
		t.Errorf("nothing should be returned or cached: got=%v upserts=%d", got, repo.upsertCalls)
	}
}

func TestSearch_HISDown_DegradesGracefully(t *testing.T) {
	repo := &fakePatientRepo{}
	his := &fakeHIS{err: errors.New("connection refused")}
	svc := newPatientSvc(repo, his, hospitalWithHIS())

	got, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{NationalID: "999"})
	if err != nil {
		t.Fatalf("HIS outage must not fail the search: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %+v", got)
	}
}

func TestSearch_InvalidHISPayload_IsIgnored(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*hospitalapi.RemotePatient)
	}{
		{"missing patient_hn", func(p *hospitalapi.RemotePatient) { p.PatientHN = "" }},
		{"invalid gender", func(p *hospitalapi.RemotePatient) { p.Gender = "X" }},
		{"invalid date_of_birth", func(p *hospitalapi.RemotePatient) { p.DateOfBirth = "17/05/1990" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rp := remote("HN9", "123")
			tc.mut(rp)
			repo := &fakePatientRepo{}
			svc := newPatientSvc(repo, &fakeHIS{patient: rp}, hospitalWithHIS())

			got, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{NationalID: "123"})
			if err != nil || len(got) != 0 || repo.upsertCalls != 0 {
				t.Fatalf("bad payload must be ignored: got=%v err=%v upserts=%d", got, err, repo.upsertCalls)
			}
		})
	}
}

func TestSearch_HospitalWithoutHIS_SkipsLookup(t *testing.T) {
	h := &model.Hospital{ID: 2, Code: "hospital-b", APIBaseURL: ""}
	his := &fakeHIS{patient: remote("HN9", "123")}
	svc := newPatientSvc(&fakePatientRepo{}, his, h)

	got, err := svc.Search(context.Background(), 2, model.PatientSearchFilter{NationalID: "123"})
	if err != nil || len(got) != 0 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	if his.calls != 0 {
		t.Error("a hospital with no HIS url must never trigger an HTTP call")
	}
}

func TestSearch_RepositoryErrors(t *testing.T) {
	t.Run("search fails", func(t *testing.T) {
		svc := newPatientSvc(&fakePatientRepo{searchErr: errors.New("boom")}, &fakeHIS{}, hospitalWithHIS())
		if _, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{}); err == nil {
			t.Fatal("expected an error")
		}
	})
	t.Run("cache write fails", func(t *testing.T) {
		repo := &fakePatientRepo{upsertErr: errors.New("boom")}
		svc := newPatientSvc(repo, &fakeHIS{patient: remote("HN9", "123")}, hospitalWithHIS())
		if _, err := svc.Search(context.Background(), 1, model.PatientSearchFilter{NationalID: "123"}); err == nil {
			t.Fatal("expected an error")
		}
	})
}
