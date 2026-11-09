package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hunterwilliams/familyfinance/internal/config"
	"github.com/hunterwilliams/familyfinance/internal/handler"
	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil, config.Config{SkipAuth: true})
	h.Register(r)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]string
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

func TestAuthMiddlewareRequiresJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil, config.Config{JWTSecret: "secret", SkipAuth: false})
	h.Register(r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/members", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDevLoginDisabledWithoutSkip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil, config.Config{SkipAuth: false})
	h.Register(r)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/dev-login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestBulkUpdateValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.New(nil, config.Config{SkipAuth: true})
	h.Register(r)

	body := bytes.NewBufferString(`{"updates":[]}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/transactions/bulk", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
