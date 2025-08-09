package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Hasher computes HMAC-SHA256 hashes using a secret key.
type Hasher struct {
	key string
}

// New creates a Hasher with the given key.
func New(key string) *Hasher {
	return &Hasher{key: key}
}

// Hash computes the HMAC-SHA256 hash of data using the Hasher's key.
// If the key is empty, it returns the HMAC of data with an empty key.
func (h *Hasher) Hash(data []byte) string {
	mac := hmac.New(sha256.New, []byte(h.key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}
