package http

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestHashMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHasher := NewMockHasher(ctrl)

	const header = "X-Hash"

	testBody := []byte("test body")
	testHash := "hash_of_test_body"
	responseBody := []byte("response body")
	responseHash := "hash_of_response_body"

	// Middleware with hasher
	middleware := HashMiddleware(mockHasher, header)

	// Handler echoes "response body"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(responseBody)
	})

	t.Run("no hasher - calls next directly", func(t *testing.T) {
		mw := HashMiddleware(nil, header)
		rec := httptest.NewRecorder()

		req := httptest.NewRequest("POST", "/", bytes.NewReader(testBody))
		req.Header.Set(header, testHash)

		mw(handler).ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, string(responseBody), rec.Body.String())
	})

	t.Run("valid hash in request - calls hasher.Hash for request and response", func(t *testing.T) {
		mockHasher.EXPECT().Hash(testBody).Return(testHash).Times(1)
		mockHasher.EXPECT().Hash(responseBody).Return(responseHash).Times(1)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/", bytes.NewReader(testBody))
		req.Header.Set(header, testHash)

		middleware(handler).ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, responseBody, rec.Body.Bytes())
		require.Equal(t, responseHash, rec.Header().Get(header))
	})

	t.Run("no hash in request header - skips request hash validation, computes response hash", func(t *testing.T) {
		mockHasher.EXPECT().Hash(responseBody).Return(responseHash).Times(1)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/", bytes.NewReader(testBody))
		// no header set

		middleware(handler).ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, responseBody, rec.Body.Bytes())
		require.Equal(t, responseHash, rec.Header().Get(header))
	})

	t.Run("invalid hash in request - returns 400", func(t *testing.T) {
		mockHasher.EXPECT().Hash(testBody).Return(testHash).Times(1)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/", bytes.NewReader(testBody))
		req.Header.Set(header, "invalidhash")

		middleware(handler).ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("error reading body - returns 400", func(t *testing.T) {
		// Create a request with a broken Body that always errors
		brokenBody := &errorReader{}
		req := httptest.NewRequest("POST", "/", brokenBody)
		rec := httptest.NewRecorder()

		middleware(handler).ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// errorReader simulates a Read error for testing.
type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func (e *errorReader) Close() error {
	return nil
}
