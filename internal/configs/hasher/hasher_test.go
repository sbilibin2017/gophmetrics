package hasher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasher_Hash(t *testing.T) {
	// Test with non-empty key
	key := "secret"
	h := New(key)
	data := []byte("test data")

	result := h.Hash(data)
	require.NotEmpty(t, result)

	// Hash should be consistent for same inputs
	result2 := h.Hash(data)
	require.Equal(t, result, result2)

	// Different data produces different hash
	result3 := h.Hash([]byte("different data"))
	require.NotEqual(t, result, result3)
}

func TestHasher_Hash_EmptyKey(t *testing.T) {
	h := New("")
	data := []byte("test data")

	result := h.Hash(data)
	require.NotEmpty(t, result)

	// Hash with empty key is deterministic
	expectedHasher := New("")
	expected := expectedHasher.Hash(data)
	require.Equal(t, expected, result)
}
