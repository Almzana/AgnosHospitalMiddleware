package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/your-username/agnos-middleware/internal/auth"
	"github.com/your-username/agnos-middleware/internal/middleware"
)

// NewRouter registers every route. It is separate from main so tests can build the same router.
func NewRouter(staff *StaffHandler, patient *PatientHandler, tokens *auth.TokenManager) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	s := r.Group("/staff")
	{
		s.POST("/create", staff.Create)
		s.POST("/login", staff.Login)
	}

	p := r.Group("/patient", middleware.RequireAuth(tokens))
	{
		p.POST("/search", patient.Search)
	}
	return r
}
