package http

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/golang/mock/gomock"
	"github.com/sbilibin2017/gophmetrics/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockRoundTripper simulates HTTP responses for resty client.
type mockRoundTripper struct {
	statusCode int
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       http.NoBody,
	}, nil
}

// headerCheckRoundTripper checks the X-Real-IP header in the request.
type headerCheckRoundTripper struct {
	expectedIP string
	statusCode int
}

func (h *headerCheckRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("X-Real-IP") != h.expectedIP {
		return nil, fmt.Errorf("missing or wrong X-Real-IP header: got %q, want %q", req.Header.Get("X-Real-IP"), h.expectedIP)
	}
	return &http.Response{
		StatusCode: h.statusCode,
		Body:       http.NoBody,
	}, nil
}

// badMetric forces JSON marshal error.
type badMetric struct{}

func (b badMetric) MarshalJSON() ([]byte, error) {
	return nil, errors.New("forced marshal error")
}

func TestMetricHTTPFacade_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCompressor := NewMockCompressor(ctrl)
	mockHasher := NewMockHasher(ctrl)
	mockCryptor := NewMockCryptor(ctrl)

	delta := int64(10)
	value := 123.456
	metrics := []*models.Metrics{
		{
			ID:        "metric1",
			MType:     "counter",
			Delta:     &delta,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "metric2",
			MType:     "gauge",
			Value:     &value,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	tests := []struct {
		name           string
		compressErr    error
		encryptErr     error
		hashReturn     string
		useHasher      bool
		useCryptor     bool
		httpStatusCode int
		expectError    bool
		metricsInput   interface{} // allows flexibility if needed
	}{
		{
			name:           "success no encryption no hash",
			useHasher:      false,
			useCryptor:     false,
			compressErr:    nil,
			httpStatusCode: 200,
			expectError:    false,
			metricsInput:   metrics,
		},
		{
			name:           "success with encryption and hash",
			useHasher:      true,
			useCryptor:     true,
			compressErr:    nil,
			encryptErr:     nil,
			hashReturn:     "fakehash",
			httpStatusCode: 200,
			expectError:    false,
			metricsInput:   metrics,
		},
		{
			name:         "compress error",
			compressErr:  errors.New("compress failed"),
			expectError:  true,
			metricsInput: metrics,
		},
		{
			name:         "encrypt error",
			useCryptor:   true,
			compressErr:  nil,
			encryptErr:   errors.New("encrypt failed"),
			expectError:  true,
			metricsInput: metrics,
		},
		{
			name:           "http error response",
			compressErr:    nil,
			httpStatusCode: 500,
			expectError:    true,
			metricsInput:   metrics,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := resty.New()
			// Use headerCheckRoundTripper to verify X-Real-IP header in every request
			client.SetTransport(&headerCheckRoundTripper{
				expectedIP: "127.0.0.1",
				statusCode: tt.httpStatusCode,
			})

			if tt.compressErr != nil {
				mockCompressor.EXPECT().Compress(gomock.Any()).Return(nil, tt.compressErr).Times(1)
			} else {
				mockCompressor.EXPECT().Compress(gomock.Any()).Return([]byte("compressed"), nil).Times(1)
			}

			if tt.useCryptor && tt.compressErr == nil {
				mockCryptor.EXPECT().Encrypt([]byte("compressed")).Return([]byte("encrypted"), tt.encryptErr).Times(1)
			}

			if tt.useHasher {
				mockHasher.EXPECT().Hash(gomock.Any()).Return(tt.hashReturn).Times(1)
			}

			var hasher Hasher
			if tt.useHasher {
				hasher = mockHasher
			}

			var cryptor Cryptor
			if tt.useCryptor {
				cryptor = mockCryptor
			}

			facade := NewMetricHTTPFacade(
				client,
				mockCompressor,
				hasher,
				cryptor,
				"key",       // dummy key
				"X-Hash",    // header name for hash
				"/update/",  // endpoint
				"127.0.0.1", // IP адрес агента
			)

			m, ok := tt.metricsInput.([]*models.Metrics)
			if !ok {
				t.Fatalf("invalid type for metricsInput")
			}

			err := facade.Update(context.Background(), m)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricHTTPFacade_Update_PostError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCompressor := NewMockCompressor(ctrl)

	delta := int64(10)
	metrics := []*models.Metrics{
		{
			ID:    "metric1",
			MType: "counter",
			Delta: &delta,
		},
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	ln.Close() // Закрываем, чтобы вызвать ошибку подключения

	client := resty.New()
	client.SetBaseURL("http://" + ln.Addr().String()) // некорректный адрес сервера

	mockCompressor.EXPECT().Compress(gomock.Any()).Return([]byte("compressed"), nil).Times(1)

	facade := NewMetricHTTPFacade(
		client,
		mockCompressor,
		nil, // no hasher
		nil, // no cryptor
		"",
		"",
		"/update",
		"127.0.0.1", // IP адрес агента
	)

	err = facade.Update(context.Background(), metrics)
	assert.Error(t, err)
}
