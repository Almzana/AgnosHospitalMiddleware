package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/your-username/agnos-middleware/internal/auth"
	"github.com/your-username/agnos-middleware/internal/model"
)

// ---------- fakes ----------

type fakeAuthService struct {
	staff    *model.Staff
	hospital *model.Hospital
	token    string
	err      error

	gotUser, gotPass, gotHospital string
}

func (f *fakeAuthService) CreateStaff(_ context.Context, u, p, h string) (*model.Staff, *model.Hospital, error) {
	f.gotUser, f.gotPass, f.gotHospital = u, p, h
	return f.staff, f.hospital, f.err
}

func (f *fakeAuthService) Login(_ context.Context, u, p, h string) (string, time.Time, error) {
	f.gotUser, f.gotPass, f.gotHospital = u, p, h
	return f.token, time.Now().Add(time.Hour), f.err
}

type fakePatientService struct {
	patients []model.Patient
	err      error

	gotHospital uint
	gotFilter   model.PatientSearchFilter
	calls       int
}

func (f *fakePatientService) Search(_ context.Context, hospitalID uint, filter model.PatientSearchFilter) ([]model.Patient, error) {
	f.calls++
	f.gotHospital, f.gotFilter = hospitalID, filter
	return f.patients, f.err
}

// ---------- helpers ----------

type env struct {
	router   *gin.Engine
	auth     *fakeAuthService
	patients *fakePatientService
	tokens   *auth.TokenManager
}

func newEnv(t *testing.T) *env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	e := &env{
		auth:     &fakeAuthService{},
		patients: &fakePatientService{},
		tokens:   auth.NewTokenManager("unit-test-secret-0123456789", time.Hour),
	}
	e.router = NewRouter(NewStaffHandler(e.auth, logger), NewPatientHandler(e.patients, logger), e.tokens)
	return e
}

func (e *env) do(method, path, body, bearer string) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func (e *env) tokenFor(hospitalID uint) string {
	tok, _, err := e.tokens.Issue(&model.Staff{ID: 1, Username: "alice", HospitalID: hospitalID})
	if err != nil {
		panic(err)
	}
	return tok
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("response is not JSON: %v (%s)", err, w.Body.String())
	}
	return m
}

const validStaffBody = `{"username":"alice","password":"s3cret-pass","hospital":"hospital-a"}`

// ---------- /health ----------

func TestHealth(t *testing.T) {
	e := newEnv(t)
	if w := e.do(http.MethodGet, "/health", "", ""); w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

// ---------- /staff/create ----------

func TestStaffCreate_Success(t *testing.T) {
	e := newEnv(t)
	e.auth.staff = &model.Staff{ID: 42, Username: "alice"}
	e.auth.hospital = &model.Hospital{Code: "hospital-a"}

	w := e.do(http.MethodPost, "/staff/create", validStaffBody, "")

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d (%s)", w.Code, w.Body.String())
	}
	body := decode(t, w)
	if body["id"] != float64(42) || body["username"] != "alice" || body["hospital"] != "hospital-a" {
		t.Errorf("unexpected body: %v", body)
	}
	if _, leaked := body["password"]; leaked {
		t.Error("response must never include the password")
	}
	if e.auth.gotUser != "alice" || e.auth.gotHospital != "hospital-a" {
		t.Errorf("service received wrong input: %q %q", e.auth.gotUser, e.auth.gotHospital)
	}
}

func TestStaffCreate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		svcErr  error
		want    int
		callSvc bool
	}{
		{"invalid json", `{not json`, nil, http.StatusBadRequest, false},
		{"empty body", ``, nil, http.StatusBadRequest, false},
		{"missing username", `{"password":"s3cret-pass","hospital":"hospital-a"}`, nil, http.StatusBadRequest, false},
		{"missing password", `{"username":"alice","hospital":"hospital-a"}`, nil, http.StatusBadRequest, false},
		{"missing hospital", `{"username":"alice","password":"s3cret-pass"}`, nil, http.StatusBadRequest, false},
		{"password too short", `{"username":"alice","password":"short","hospital":"hospital-a"}`, nil, http.StatusBadRequest, false},
		{"username too short", `{"username":"al","password":"s3cret-pass","hospital":"hospital-a"}`, nil, http.StatusBadRequest, false},
		{"unknown hospital", validStaffBody, model.ErrHospitalNotFound, http.StatusBadRequest, true},
		{"duplicate staff", validStaffBody, model.ErrStaffExists, http.StatusConflict, true},
		{"unexpected failure", validStaffBody, errors.New("db down"), http.StatusInternalServerError, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			e.auth.err = tc.svcErr

			w := e.do(http.MethodPost, "/staff/create", tc.body, "")

			if w.Code != tc.want {
				t.Fatalf("want %d, got %d (%s)", tc.want, w.Code, w.Body.String())
			}
			if _, ok := decode(t, w)["error"]; !ok {
				t.Error("error responses must carry an \"error\" field")
			}
			if !tc.callSvc && e.auth.gotUser != "" {
				t.Error("service must not be called for invalid input")
			}
		})
	}
}

func TestStaffCreate_InternalErrorDoesNotLeakDetails(t *testing.T) {
	e := newEnv(t)
	e.auth.err = errors.New("pq: password authentication failed for user admin")

	w := e.do(http.MethodPost, "/staff/create", validStaffBody, "")

	if bytes.Contains(w.Body.Bytes(), []byte("pq:")) {
		t.Errorf("internal error text leaked to the client: %s", w.Body.String())
	}
}

// ---------- /staff/login ----------

func TestStaffLogin_Success(t *testing.T) {
	e := newEnv(t)
	e.auth.token = "jwt-token"

	w := e.do(http.MethodPost, "/staff/login", validStaffBody, "")

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", w.Code, w.Body.String())
	}
	body := decode(t, w)
	if body["token"] != "jwt-token" {
		t.Errorf("token missing: %v", body)
	}
	if _, ok := body["expires_at"]; !ok {
		t.Error("expires_at missing")
	}
}

func TestStaffLogin_Errors(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		svcErr error
		want   int
	}{
		{"invalid json", `oops`, nil, http.StatusBadRequest},
		{"missing hospital", `{"username":"alice","password":"s3cret-pass"}`, nil, http.StatusBadRequest},
		{"missing password", `{"username":"alice","hospital":"hospital-a"}`, nil, http.StatusBadRequest},
		{"bad credentials", validStaffBody, model.ErrInvalidCredentials, http.StatusUnauthorized},
		{"unexpected failure", validStaffBody, errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t)
			e.auth.err = tc.svcErr

			w := e.do(http.MethodPost, "/staff/login", tc.body, "")

			if w.Code != tc.want {
				t.Fatalf("want %d, got %d (%s)", tc.want, w.Code, w.Body.String())
			}
			if _, hasToken := decode(t, w)["token"]; hasToken {
				t.Error("no token may be returned on failure")
			}
		})
	}
}

// ---------- /patient/search ----------

func TestPatientSearch_RequiresLogin(t *testing.T) {
	e := newEnv(t)
	good := e.tokenFor(1)

	tests := map[string]string{
		"no token":      "",
		"garbage token": "abc.def.ghi",
		"tampered":      good + "x",
	}
	for name, bearer := range tests {
		t.Run(name, func(t *testing.T) {
			w := e.do(http.MethodPost, "/patient/search", `{}`, bearer)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("want 401, got %d", w.Code)
			}
			if e.patients.calls != 0 {
				t.Error("service must not be reached without authentication")
			}
		})
	}
}

func TestPatientSearch_Success(t *testing.T) {
	e := newEnv(t)
	dob := time.Date(1990, 5, 17, 0, 0, 0, 0, time.UTC)
	e.patients.patients = []model.Patient{{
		PatientHN: "HN1", FirstNameEN: "Somchai", LastNameEN: "Jaidee", FirstNameTH: "สมชาย",
		NationalID: "1234567890123", DateOfBirth: &dob, Gender: "M", Email: "s@example.com",
	}}

	w := e.do(http.MethodPost, "/patient/search",
		`{"national_id":"1234567890123","first_name":"Somchai","date_of_birth":"1990-05-17","phone_number":"0812345678","email":"s@example.com"}`,
		e.tokenFor(9))

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d (%s)", w.Code, w.Body.String())
	}
	body := decode(t, w)
	if body["count"] != float64(1) {
		t.Errorf("count: %v", body["count"])
	}
	first := body["patients"].([]any)[0].(map[string]any)
	if first["patient_hn"] != "HN1" || first["date_of_birth"] != "1990-05-17" || first["gender"] != "M" || first["first_name_th"] != "สมชาย" {
		t.Errorf("unexpected patient payload: %v", first)
	}

	f := e.patients.gotFilter
	if f.NationalID != "1234567890123" || f.FirstName != "Somchai" || f.PhoneNumber != "0812345678" || f.Email != "s@example.com" {
		t.Errorf("filter not mapped: %+v", f)
	}
	if f.DateOfBirth == nil || f.DateOfBirth.Format("2006-01-02") != "1990-05-17" {
		t.Errorf("date_of_birth not parsed: %v", f.DateOfBirth)
	}
}

func TestPatientSearch_HospitalComesFromTokenOnly(t *testing.T) {
	e := newEnv(t)

	// the client tries to smuggle in another hospital
	w := e.do(http.MethodPost, "/patient/search", `{"hospital_id":99,"hospital":"hospital-b","national_id":"1"}`, e.tokenFor(4))

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	if e.patients.gotHospital != 4 {
		t.Fatalf("search must be scoped to the token's hospital (4), got %d", e.patients.gotHospital)
	}
}

func TestPatientSearch_AllFieldsOptional(t *testing.T) {
	for name, reqBody := range map[string]string{"empty object": `{}`, "no body": ``} {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			w := e.do(http.MethodPost, "/patient/search", reqBody, e.tokenFor(1))
			if w.Code != http.StatusOK {
				t.Fatalf("want 200, got %d (%s)", w.Code, w.Body.String())
			}
			resp := decode(t, w)
			if resp["count"] != float64(0) {
				t.Errorf("count: %v", resp["count"])
			}
			if list, ok := resp["patients"].([]any); !ok || len(list) != 0 {
				t.Errorf("patients must be an empty array, not null: %v", resp["patients"])
			}
		})
	}
}

func TestPatientSearch_BadRequests(t *testing.T) {
	tests := map[string]string{
		"invalid json":         `{bad`,
		"wrong dob format":     `{"date_of_birth":"17/05/1990"}`,
		"impossible dob":       `{"date_of_birth":"1990-02-31"}`,
		"wrong type for field": `{"national_id":123}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			w := e.do(http.MethodPost, "/patient/search", body, e.tokenFor(1))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("want 400, got %d (%s)", w.Code, w.Body.String())
			}
			if e.patients.calls != 0 {
				t.Error("service must not be called for invalid input")
			}
		})
	}
}

func TestPatientSearch_ServiceError(t *testing.T) {
	e := newEnv(t)
	e.patients.err = errors.New("db down")

	w := e.do(http.MethodPost, "/patient/search", `{}`, e.tokenFor(1))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
	if bytes.Contains(w.Body.Bytes(), []byte("db down")) {
		t.Error("internal error text leaked to the client")
	}
}
