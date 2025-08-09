package compressor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressor_CompressDecompress(t *testing.T) {
	c := NewCompressor()
	original := []byte("Hello, gzip compression!")

	compressed, err := c.Compress(original)
	require.NoError(t, err)
	require.NotEmpty(t, compressed)
	require.NotEqual(t, original, compressed)

	decompressed, err := c.Decompress(compressed)
	require.NoError(t, err)
	require.Equal(t, original, decompressed)
}

func TestCompressor_DecompressInvalidData(t *testing.T) {
	c := NewCompressor()
	invalidData := []byte("not gzip data")

	_, err := c.Decompress(invalidData)
	require.Error(t, err)
}

func TestCompressor_CompressEmptyData(t *testing.T) {
	c := NewCompressor()
	empty := []byte("")

	compressed, err := c.Compress(empty)
	require.NoError(t, err)
	require.NotEmpty(t, compressed)

	decompressed, err := c.Decompress(compressed)
	require.NoError(t, err)
	require.Equal(t, empty, decompressed)
}
