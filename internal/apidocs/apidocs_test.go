package apidocs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r)

	cases := map[string]string{
		"/swagger":              "swagger-ui",
		"/swagger/openapi.yaml": "/patient/search",
	}
	for path, want := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d", path, w.Code)
		}
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("%s: body does not contain %q", path, want)
		}
	}
}

func TestSpecIsValidYAML(t *testing.T) {
	var doc struct {
		OpenAPI string                    `yaml:"openapi"`
		Paths   map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	for _, p := range []string{"/health", "/staff/create", "/staff/login", "/patient/search"} {
		if _, ok := doc.Paths[p]; !ok {
			t.Errorf("spec is missing path %s", p)
		}
	}
}
