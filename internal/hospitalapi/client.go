// Package hospitalapi is the client for a hospital's HIS (Hospital Information System) REST API.
package hospitalapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Almzaba/AgnosHospitalMiddleware/internal/model"
)

// RemotePatient is the JSON body returned by GET {base}/patient/search/{id}.
type RemotePatient struct {
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

// Client fetches patients from a HIS.
type Client interface {
	FetchPatient(ctx context.Context, baseURL, id string) (*RemotePatient, error)
}

type httpClient struct{ http *http.Client }

// NewClient returns a Client using the given request timeout.
func NewClient(timeout time.Duration) Client {
	return &httpClient{http: &http.Client{Timeout: timeout}}
}

// FetchPatient calls the HIS. It returns model.ErrPatientNotFound on HTTP 404.
func (c *httpClient) FetchPatient(ctx context.Context, baseURL, id string) (*RemotePatient, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/patient/search/" + url.PathEscape(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build HIS request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call HIS: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, model.ErrPatientNotFound
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("HIS returned status %d", resp.StatusCode)
	}

	var p RemotePatient
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("decode HIS response: %w", err)
	}
	return &p, nil
}
