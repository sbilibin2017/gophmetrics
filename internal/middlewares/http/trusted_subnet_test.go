package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrustedSubnetMiddleware(t *testing.T) {
	const validCIDR = "192.168.1.0/24"

	// handler to test if next was called
	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	t.Run("No subnet - passes through", func(t *testing.T) {
		nextCalled = false
		mw := TrustedSubnetMiddleware("")
		handler := mw(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		resp := w.Result()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, nextCalled)
	})

	t.Run("Valid subnet - allowed IP", func(t *testing.T) {
		nextCalled = false
		mw := TrustedSubnetMiddleware(validCIDR)
		handler := mw(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "192.168.1.100")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		resp := w.Result()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, nextCalled)
	})

	t.Run("Valid subnet - disallowed IP", func(t *testing.T) {
		nextCalled = false
		mw := TrustedSubnetMiddleware(validCIDR)
		handler := mw(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		resp := w.Result()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		assert.False(t, nextCalled)
	})

	t.Run("Missing X-Real-IP header", func(t *testing.T) {
		nextCalled = false
		mw := TrustedSubnetMiddleware(validCIDR)
		handler := mw(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		resp := w.Result()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		assert.False(t, nextCalled)
	})

	t.Run("Invalid CIDR - always 500", func(t *testing.T) {
		mw := TrustedSubnetMiddleware("invalid_cidr")
		handler := mw(nextHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "192.168.1.100")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
		resp := w.Result()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}
