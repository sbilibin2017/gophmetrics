package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithAddress(t *testing.T) {
	cfg := &ServerConfig{}
	opt := WithAddress("", "  ", "localhost:9090")
	err := opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "localhost:9090", cfg.Address)

	// No valid address, should keep default empty
	cfg = &ServerConfig{}
	opt = WithAddress("", "  ")
	err = opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "", cfg.Address)
}

func TestWithStoreInterval(t *testing.T) {
	cfg := &ServerConfig{}
	opt := WithStoreInterval(0, -1, 300)
	err := opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 300, cfg.StoreInterval)

	// No positive interval, should keep default 0
	cfg = &ServerConfig{}
	opt = WithStoreInterval(0, -5)
	err = opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 0, cfg.StoreInterval)
}

func TestWithFileStoragePath(t *testing.T) {
	cfg := &ServerConfig{}
	opt := WithFileStoragePath("", "  ", "/tmp/metrics.json")
	err := opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "/tmp/metrics.json", cfg.FileStoragePath)

	// No valid path, should keep default empty
	cfg = &ServerConfig{}
	opt = WithFileStoragePath("", "  ")
	err = opt(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "", cfg.FileStoragePath)
}

func TestWithRestore(t *testing.T) {
	cfg := &ServerConfig{}
	opt := WithRestore(false, false, true)
	err := opt(cfg)
	assert.NoError(t, err)
	assert.True(t, cfg.Restore)

	// No true values, should keep default false
	cfg = &ServerConfig{}
	opt = WithRestore(false, false)
	err = opt(cfg)
	assert.NoError(t, err)
	assert.False(t, cfg.Restore)
}

func TestNewServerConfig(t *testing.T) {
	cfg, err := NewServerConfig(
		WithAddress("localhost:9090"),
		WithStoreInterval(200),
		WithFileStoragePath("/tmp/data.json"),
		WithRestore(true),
	)
	assert.NoError(t, err)
	assert.Equal(t, "localhost:9090", cfg.Address)
	assert.Equal(t, 200, cfg.StoreInterval)
	assert.Equal(t, "/tmp/data.json", cfg.FileStoragePath)
	assert.True(t, cfg.Restore)
}
