// Package handler contains the HTTP layer: parsing, validation and response mapping.
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Almzana/AgnosHospitalMiddleware/internal/model"
	"github.com/Almzana/AgnosHospitalMiddleware/internal/service"
)

// StaffHandler serves /staff/*.
type StaffHandler struct {
	auth   service.AuthService
	logger *slog.Logger
}

// NewStaffHandler builds a StaffHandler.
func NewStaffHandler(auth service.AuthService, logger *slog.Logger) *StaffHandler {
	return &StaffHandler{auth: auth, logger: logger}
}

type staffRequest struct {
	Username string `json:"username" binding:"required,min=3,max=100"`
	Password string `json:"password" binding:"required,min=8,max=72"` // bcrypt ignores bytes past 72
	Hospital string `json:"hospital" binding:"required,max=50"`
}

type staffResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Hospital string `json:"hospital"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Create handles POST /staff/create.
func (h *StaffHandler) Create(c *gin.Context) {
	var req staffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	staff, hospital, err := h.auth.CreateStaff(c.Request.Context(), req.Username, req.Password, req.Hospital)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, staffResponse{ID: staff.ID, Username: staff.Username, Hospital: hospital.Code})
	case errors.Is(err, model.ErrHospitalNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, model.ErrStaffExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		h.internalError(c, "create staff", err)
	}
}

// Login handles POST /staff/login.
func (h *StaffHandler) Login(c *gin.Context) {
	var req staffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err)
		return
	}

	token, expiresAt, err := h.auth.Login(c.Request.Context(), req.Username, req.Password, req.Hospital)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, loginResponse{Token: token, ExpiresAt: expiresAt})
	case errors.Is(err, model.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	default:
		h.internalError(c, "login", err)
	}
}

func (h *StaffHandler) internalError(c *gin.Context, op string, err error) {
	h.logger.Error(op, "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

func badRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
}
