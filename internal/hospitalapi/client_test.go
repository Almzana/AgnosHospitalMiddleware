package hospitalapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/your-username/agnos-middleware/internal/model"
)

const okBody = `{
  "first_name_th":"สมชาย","middle_name_th":"","last_name_th":"ใจดี",
  "first_name_en":"Somchai","middle_name_en":"","last_name_en":"Jaidee",
  "date_of_birth":"1990-05-17","patient_hn":"HN123",
  "national_id":"1234567890123","passport_id":"",
  "phone_number":"0812345678","email":"somchai@example.com","gender":"M"}`

func TestFetchPatient_Success(t *testing.T) {
	var gotPath, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAccept = r.URL.EscapedPath(), r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okBody))
	}))
	defer srv.Close()

	// trailing slash on the base URL must be tolerated
	p, err := NewClient(time.Second).FetchPatient(context.Background(), srv.URL+"/", "1234567890123")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/patient/search/1234567890123" {
		t.Errorf("wrong path: %s", gotPath)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept header: %q", gotAccept)
	}
	if p.PatientHN != "HN123" || p.FirstNameTH != "สมชาย" || p.Gender != "M" || p.DateOfBirth != "1990-05-17" {
		t.Errorf("response not decoded: %+v", p)
	}
}

func TestFetchPatient_EscapesID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(okBody))
	}))
	defer srv.Close()

	if _, err := NewClient(time.Second).FetchPatient(context.Background(), srv.URL, "../admin?x=1"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/patient/search/..%2Fadmin%3Fx=1" {
		t.Errorf("id was not path-escaped: %s", gotPath)
	}
}

func TestFetchPatient_Errors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantNF  bool
	}{
		{"404 maps to ErrPatientNotFound", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }, true},
		{"500", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }, false},
		{"401", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) }, false},
		{"invalid json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			p, err := NewClient(time.Second).FetchPatient(context.Background(), srv.URL, "1")
			if err == nil || p != nil {
				t.Fatalf("expected an error, got patient=%v err=%v", p, err)
			}
			if got := errors.Is(err, model.ErrPatientNotFound); got != tc.wantNF {
				t.Errorf("ErrPatientNotFound = %v, want %v (err: %v)", got, tc.wantNF, err)
			}
		})
	}
}

func TestFetchPatient_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	if _, err := NewClient(50*time.Millisecond).FetchPatient(context.Background(), srv.URL, "1"); err == nil {
		t.Fatal("expected a timeout error")
	}
}

func TestFetchPatient_UnreachableHost(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing is listening any more

	if _, err := NewClient(time.Second).FetchPatient(context.Background(), url, "1"); err == nil {
		t.Fatal("expected a connection error")
	}
}

func TestFetchPatient_InvalidBaseURL(t *testing.T) {
	if _, err := NewClient(time.Second).FetchPatient(context.Background(), "http://bad host", "1"); err == nil {
		t.Fatal("expected an error for a malformed URL")
	}
}
