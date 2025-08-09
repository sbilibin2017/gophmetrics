package compressor

import (
	"bytes"
	"compress/gzip"
	"io"
)

// Compressor provides methods to compress and decompress data using gzip.
type Compressor struct{}

// NewCompressor creates a new Compressor instance.
func NewCompressor() *Compressor {
	return &Compressor{}
}

// Compress compresses the input data using gzip.
func (c *Compressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write(data)
	if err != nil {
		_ = gzw.Close()
		return nil, err
	}
	if err = gzw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decompress decompresses the input gzip-compressed data.
func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	buf := bytes.NewReader(data)
	gzr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	var out bytes.Buffer
	_, err = io.Copy(&out, gzr)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
