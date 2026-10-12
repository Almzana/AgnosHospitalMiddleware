package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Almzana/AgnosHospitalMiddleware/internal/auth"
	"github.com/Almzana/AgnosHospitalMiddleware/internal/model"
)

func setup(t *testing.T) (*gin.Engine, *auth.TokenManager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tokens := auth.NewTokenManager("unit-test-secret-0123456789", time.Hour)

	r := gin.New()
	r.GET("/protected", RequireAuth(tokens), func(c *gin.Context) {
		claims, ok := ClaimsFrom(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"hospital_id": claims.HospitalID})
	})
	return r, tokens
}

func call(r http.Handler, header string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireAuth_ValidToken(t *testing.T) {
	r, tokens := setup(t)
	token, _, err := tokens.Issue(&model.Staff{ID: 1, Username: "alice", HospitalID: 5})
	if err != nil {
		t.Fatal(err)
	}

	for _, scheme := range []string{"Bearer ", "bearer "} { // scheme is case-insensitive
		w := call(r, scheme+token)
		if w.Code != http.StatusOK {
			t.Fatalf("%q: want 200, got %d (%s)", scheme, w.Code, w.Body.String())
		}
	}
}

func TestRequireAuth_Rejected(t *testing.T) {
	r, tokens := setup(t)
	good, _, _ := tokens.Issue(&model.Staff{ID: 1, Username: "alice", HospitalID: 5})

	tests := map[string]string{
		"no header":       "",
		"wrong scheme":    "Basic " + good,
		"no token":        "Bearer ",
		"token only":      good,
		"invalid token":   "Bearer abc.def.ghi",
		"tampered token":  "Bearer " + good + "x",
		"empty after gap": "Bearer    ",
	}
	for name, header := range tests {
		t.Run(name, func(t *testing.T) {
			if w := call(r, header); w.Code != http.StatusUnauthorized {
				t.Fatalf("want 401, got %d", w.Code)
			}
		})
	}
}
