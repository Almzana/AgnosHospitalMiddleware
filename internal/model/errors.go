package model

import "errors"

// Sentinel errors returned by repositories and services. Handlers translate them to HTTP codes.
var (
	ErrHospitalNotFound   = errors.New("hospital not found")
	ErrStaffNotFound      = errors.New("staff not found")
	ErrStaffExists        = errors.New("staff already exists in this hospital")
	ErrInvalidCredentials = errors.New("invalid username, password or hospital")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrPatientNotFound    = errors.New("patient not found in HIS")
)
