// Command mockhis is a stand-in for Hospital A's HIS API, used for local development and demos.
//
//	GET /patient/search/{id}  - id is a national_id or passport_id
//
// Known ids return a patient, unknown ids return 404 and FailingID returns 500 so the
// middleware's "HIS is down" path can be tried by hand.
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// FailingID makes the mock answer 500, simulating an HIS outage.
const FailingID = "9999999999999"

// patient is the HIS response body (same JSON contract as hospitalapi.RemotePatient).
type patient struct {
	FirstNameTH  string `json:"first_name_th"`
	MiddleNameTH string `json:"middle_name_th"`
	LastNameTH   string `json:"last_name_th"`
	FirstNameEN  string `json:"first_name_en"`
	MiddleNameEN string `json:"middle_name_en"`
	LastNameEN   string `json:"last_name_en"`
	DateOfBirth  string `json:"date_of_birth"`
	PatientHN    string `json:"patient_hn"`
	NationalID   string `json:"national_id"`
	PassportID   string `json:"passport_id"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Gender       string `json:"gender"`
}

var patients = []patient{
	{
		FirstNameTH: "สมชาย", LastNameTH: "ใจดี", FirstNameEN: "Somchai", LastNameEN: "Jaidee",
		DateOfBirth: "1990-05-17", PatientHN: "HN-A0001", NationalID: "1234567890123",
		PhoneNumber: "0812345678", Email: "somchai@example.com", Gender: "M",
	},
	{
		FirstNameTH: "สมศรี", MiddleNameTH: "มณี", LastNameTH: "รักดี",
		FirstNameEN: "Somsri", MiddleNameEN: "Manee", LastNameEN: "Rakdee",
		DateOfBirth: "1985-11-02", PatientHN: "HN-A0002", NationalID: "1103700012345",
		PhoneNumber: "0898765432", Email: "somsri@example.com", Gender: "F",
	},
	{
		// Foreigner: passport only.
		FirstNameEN: "John", LastNameEN: "Smith",
		DateOfBirth: "1978-03-21", PatientHN: "HN-A0003", PassportID: "AA1234567",
		PhoneNumber: "0861112222", Email: "john.smith@example.com", Gender: "M",
	},
	{
		// HIS sent no gender: must still be stored by the middleware.
		FirstNameTH: "มานะ", LastNameTH: "อดทน", FirstNameEN: "Mana", LastNameEN: "Odthon",
		DateOfBirth: "2001-08-30", PatientHN: "HN-A0004", NationalID: "3100500099999",
		PhoneNumber: "0823334444", Email: "mana@example.com",
	},
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	addr := ":" + envOr("PORT", "8081")

	srv := &http.Server{Addr: addr, Handler: newHandler(logger), ReadHeaderTimeout: 5 * time.Second}
	logger.Info("mock HIS listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("mock HIS stopped", "error", err)
		os.Exit(1)
	}
}

func newHandler(logger *slog.Logger) http.Handler {
	byID := make(map[string]patient)
	for _, p := range patients {
		if p.NationalID != "" {
			byID[p.NationalID] = p
		}
		if p.PassportID != "" {
			byID[p.PassportID] = p
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /patient/search/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		logger.Info("lookup", "id", id)

		if id == FailingID {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "simulated HIS failure"})
			return
		}
		p, ok := byID[id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "patient not found"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
