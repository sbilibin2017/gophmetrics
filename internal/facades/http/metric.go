package http

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/sbilibin2017/gophmetrics/internal/models"
)

type Compressor interface {
	Compress(data []byte) ([]byte, error)
}

type Hasher interface {
	Hash(data []byte) string
}

type Cryptor interface {
	Encrypt(data []byte) ([]byte, error)
}

// MetricHTTPFacade provides HTTP-based metric updates.
type MetricHTTPFacade struct {
	client     *resty.Client
	compressor Compressor
	hasher     Hasher
	cryptor    Cryptor
	header     string
	endpoint   string
	ip         string // добавлено поле для IP
}

// NewMetricHTTPFacade creates a new MetricHTTPFacade with the given REST client,
// compressor, hasher, cryptor, and optional key/header for hash header.
func NewMetricHTTPFacade(
	client *resty.Client,
	compressor Compressor,
	hasher Hasher,
	cryptor Cryptor,
	key string,
	header string,
	endpoint string,
	ip string, // IP агента передается сюда
) *MetricHTTPFacade {
	return &MetricHTTPFacade{
		client:     client,
		compressor: compressor,
		hasher:     hasher,
		cryptor:    cryptor,
		header:     header,
		endpoint:   endpoint,
		ip:         ip, // сохранили IP в структуре
	}
}

// Update sends metric updates using JSON marshaling, gzip compression,
// optional encryption, and adds hash header computed on raw JSON.
func (f *MetricHTTPFacade) Update(ctx context.Context, metrics []*models.Metrics) error {
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	compressedData, err := f.compressor.Compress(jsonData)
	if err != nil {
		return err
	}

	if f.cryptor != nil {
		compressedData, err = f.cryptor.Encrypt(compressedData)
		if err != nil {
			return err
		}
	}

	req := f.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compressedData)

	if f.ip != "" {
		req.SetHeader("X-Real-IP", f.ip)
	}

	if f.header != "" && f.hasher != nil {
		hash := f.hasher.Hash(jsonData)
		req.SetHeader(f.header, hash)
	}

	resp, err := req.Post(f.endpoint)
	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("server responded with status %d", resp.StatusCode())
	}

	return nil
}
