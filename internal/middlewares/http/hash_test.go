package http

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashMiddleware_NoKey_PassesThrough(t *testing.T) {
	handler := HashMiddleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "response body", rr.Body.String())
	assert.Empty(t, rr.Header().Get("HashSHA256"))
}

func TestHashMiddleware_BadBodyRead_ReturnsBadRequest(t *testing.T) {
	key := "secret"

	// Создадим ридер, который возвращает ошибку при чтении
	badReader := &errorReader{}

	req := httptest.NewRequest(http.MethodPost, "/", badReader)
	rr := httptest.NewRecorder()

	handler := HashMiddleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should NOT be called on body read error")
	}))

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Empty(t, rr.Body.String())
}

func TestHashMiddleware_RequestHashMismatch_BadRequest(t *testing.T) {
	key := "secret"

	body := []byte("test body")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("HashSHA256", "invalidhash")

	handler := HashMiddleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should NOT be called on hash mismatch")
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Empty(t, rr.Body.String())
}

func TestHashMiddleware_RequestHashMatch_Success(t *testing.T) {
	key := "secret"
	body := []byte("test body")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

	expectedHash := computeHash(body, key)
	req.Header.Set("HashSHA256", expectedHash)

	handler := HashMiddleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, body, data)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body"))
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "response body", rr.Body.String())

	respHash := rr.Header().Get("HashSHA256")
	expectedRespHash := computeHash([]byte("response body"), key)
	assert.Equal(t, expectedRespHash, respHash)
}

func TestHashMiddleware_ResponseHashSet(t *testing.T) {
	key := "secret"
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler := HashMiddleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body"))
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	expectedHash := computeHash([]byte("response body"), key)
	assert.Equal(t, expectedHash, rr.Header().Get("HashSHA256"))
}

// errorReader - ридер, который всегда возвращает ошибку
type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
