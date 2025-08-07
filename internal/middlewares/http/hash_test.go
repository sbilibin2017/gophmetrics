package http

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// helper to compute HMAC-SHA256 hash
func computeHMAC(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestHashMiddleware_ValidRequestAndResponseHash(t *testing.T) {
	key := "mysecretkey"
	body := []byte(`{"valid":"json"}`)
	hash := computeHMAC(body, key)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("HashSHA256", hash)

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`response-ok`))
	})

	HashMiddleware(key)(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "response-ok", rr.Body.String())

	expectedRespHash := computeHMAC([]byte("response-ok"), key)
	assert.Equal(t, expectedRespHash, rr.Header().Get("HashSHA256"))
}

func TestHashMiddleware_InvalidRequestHash(t *testing.T) {
	key := "mysecretkey"
	body := []byte(`{"invalid":"hash"}`)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("HashSHA256", "wronghashvalue")

	rr := httptest.NewRecorder()

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	HashMiddleware(key)(handler).ServeHTTP(rr, req)

	assert.False(t, called, "handler should not be called with invalid hash")
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHashMiddleware_MissingHashHeader(t *testing.T) {
	key := "mysecretkey"
	body := []byte(`{"no":"hash"}`)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok-missing-hash"))
	})

	HashMiddleware(key)(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ok-missing-hash", rr.Body.String())

	expected := computeHMAC([]byte("ok-missing-hash"), key)
	assert.Equal(t, expected, rr.Header().Get("HashSHA256"))
}

func TestHashMiddleware_EmptyKey_SkipsMiddleware(t *testing.T) {
	body := []byte(`{"skip":"middleware"}`)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte("skipped"))
	})

	HashMiddleware("")(handler).ServeHTTP(rr, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "skipped", rr.Body.String())
	assert.Empty(t, rr.Header().Get("HashSHA256"))
}

func TestHashMiddleware_BrokenBody(t *testing.T) {
	key := "mysecretkey"

	req := httptest.NewRequest(http.MethodPost, "/", brokenReader{})
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when body is unreadable")
	})

	HashMiddleware(key)(handler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

// brokenReader always returns error when read
type brokenReader struct{}

func (brokenReader) Read(p []byte) (n int, err error) {
	return 0, assert.AnError
}

func (brokenReader) Close() error {
	return nil
}
