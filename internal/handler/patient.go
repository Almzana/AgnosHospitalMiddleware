package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Almzaba/AgnosHospitalMiddleware/internal/middleware"
	"github.com/Almzaba/AgnosHospitalMiddleware/internal/model"
	"github.com/Almzaba/AgnosHospitalMiddleware/internal/service"
)

const dateLayout = "2006-01-02"

// PatientHandler serves /patient/*.
type PatientHandler struct {
	patients service.PatientService
	logger   *slog.Logger
}

// NewPatientHandler builds a PatientHandler.
func NewPatientHandler(patients service.PatientService, logger *slog.Logger) *PatientHandler {
	return &PatientHandler{patients: patients, logger: logger}
}

// searchRequest: every field is optional.
type searchRequest struct {
	NationalID  string `json:"national_id"`
	PassportID  string `json:"passport_id"`
	FirstName   string `json:"first_name"`
	MiddleName  string `json:"middle_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"` // YYYY-MM-DD
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
}

type patientResponse struct {
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

// Search handles POST /patient/search (JWT required).
// POST with a JSON body is used so personal data never ends up in URLs or access logs.
func (h *PatientHandler) Search(c *gin.Context) {
	claims, ok := middleware.ClaimsFrom(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var req searchRequest
	// An empty body is valid: every field is optional.
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, err)
			return
		}
	}

	filter := model.PatientSearchFilter{
		NationalID:  req.NationalID,
		PassportID:  req.PassportID,
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
	}
	if req.DateOfBirth != "" {
		dob, err := time.Parse(dateLayout, req.DateOfBirth)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date_of_birth must be in YYYY-MM-DD format"})
			return
		}
		filter.DateOfBirth = &dob
	}

	// The hospital comes from the token, never from the request body.
	patients, err := h.patients.Search(c.Request.Context(), claims.HospitalID, filter)
	if err != nil {
		h.logger.Error("search patients", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	out := make([]patientResponse, 0, len(patients))
	for _, p := range patients {
		out = append(out, toResponse(p))
	}
	c.JSON(http.StatusOK, gin.H{"count": len(out), "patients": out})
}

func toResponse(p model.Patient) patientResponse {
	dob := ""
	if p.DateOfBirth != nil {
		dob = p.DateOfBirth.Format(dateLayout)
	}
	return patientResponse{
		FirstNameTH:  p.FirstNameTH,
		MiddleNameTH: p.MiddleNameTH,
		LastNameTH:   p.LastNameTH,
		FirstNameEN:  p.FirstNameEN,
		MiddleNameEN: p.MiddleNameEN,
		LastNameEN:   p.LastNameEN,
		DateOfBirth:  dob,
		PatientHN:    p.PatientHN,
		NationalID:   p.NationalID,
		PassportID:   p.PassportID,
		PhoneNumber:  p.PhoneNumber,
		Email:        p.Email,
		Gender:       p.Gender,
	}
}
