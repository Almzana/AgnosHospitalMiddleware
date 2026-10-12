package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Almzaba/AgnosHospitalMiddleware/internal/hospitalapi"
	"github.com/Almzaba/AgnosHospitalMiddleware/internal/model"
)

// The mock is exercised through the real HIS client, which proves both speak the same contract.
func TestMockHISWithRealClient(t *testing.T) {
	srv := httptest.NewServer(newHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	defer srv.Close()
	client := hospitalapi.NewClient(time.Second)
	ctx := context.Background()

	t.Run("by national id", func(t *testing.T) {
		p, err := client.FetchPatient(ctx, srv.URL, "1234567890123")
		if err != nil {
			t.Fatal(err)
		}
		if p.PatientHN != "HN-A0001" || p.FirstNameTH != "สมชาย" {
			t.Errorf("unexpected patient %+v", p)
		}
	})

	t.Run("by passport id", func(t *testing.T) {
		p, err := client.FetchPatient(ctx, srv.URL, "AA1234567")
		if err != nil {
			t.Fatal(err)
		}
		if p.PatientHN != "HN-A0003" {
			t.Errorf("unexpected patient %+v", p)
		}
	})

	t.Run("unknown id is not found", func(t *testing.T) {
		if _, err := client.FetchPatient(ctx, srv.URL, "0000000000000"); err != model.ErrPatientNotFound {
			t.Errorf("want ErrPatientNotFound, got %v", err)
		}
	})

	t.Run("failing id is a server error", func(t *testing.T) {
		_, err := client.FetchPatient(ctx, srv.URL, FailingID)
		if err == nil || err == model.ErrPatientNotFound {
			t.Errorf("want a non-404 error, got %v", err)
		}
	})
}

func TestOnlyGETIsServed(t *testing.T) {
	h := newHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/patient/search/1234567890123", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}
