package server

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

// reinitFlags re-registers flags exactly as done in init()
func reinitFlags() {
	pflag.CommandLine = pflag.NewFlagSet("test", pflag.ContinueOnError)

	pflag.StringVarP(&addr, "address", "a", "localhost:8080", "server URL")
	pflag.IntVarP(&storeInterval, "interval", "i", 300, "interval in seconds to save metrics (0 = sync save)")
	pflag.StringVarP(&fileStoragePath, "file", "f", "metrics.json", "file path to store metrics")
	pflag.BoolVarP(&restore, "restore", "r", true, "restore metrics from file on startup")
	pflag.StringVarP(&databaseDSN, "database-dsn", "d", "", "PostgreSQL DSN connection string")
}

// resetFlagsAndEnv resets flags, backing variables, and environment variables
func resetFlagsAndEnv() {
	// Rebind all flags
	reinitFlags()

	// Clear env vars
	os.Unsetenv("ADDRESS")
	os.Unsetenv("STORE_INTERVAL")
	os.Unsetenv("FILE_STORAGE_PATH")
	os.Unsetenv("RESTORE")
	os.Unsetenv("DATABASE_DSN")

	// Reset os.Args to default
	os.Args = []string{"cmd"}

	// Reset global flag-bound variables
	addr = "localhost:8080"
	storeInterval = 300
	fileStoragePath = "metrics.json"
	restore = true
	databaseDSN = ""
}

func TestNewConfig_Defaults(t *testing.T) {
	resetFlagsAndEnv()

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, "localhost:8080", cfg.Addr)
	assert.Equal(t, 300, cfg.StoreInterval)
	assert.Equal(t, "metrics.json", cfg.FileStoragePath)
	assert.True(t, cfg.Restore)
	assert.Equal(t, "", cfg.DatabaseDSN)
	assert.Equal(t, "migrations", cfg.MigrationsDir)
}

func TestNewConfig_EnvOverrides(t *testing.T) {
	resetFlagsAndEnv()

	os.Setenv("ADDRESS", "127.0.0.1:9090")
	os.Setenv("STORE_INTERVAL", "100")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/metrics.json")
	os.Setenv("RESTORE", "false")
	os.Setenv("DATABASE_DSN", "user=foo password=bar dbname=baz sslmode=disable")

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9090", cfg.Addr)
	assert.Equal(t, 100, cfg.StoreInterval)
	assert.Equal(t, "/tmp/metrics.json", cfg.FileStoragePath)
	assert.False(t, cfg.Restore)
	assert.Equal(t, "user=foo password=bar dbname=baz sslmode=disable", cfg.DatabaseDSN)
}

func TestNewConfig_InvalidStoreIntervalEnv(t *testing.T) {
	resetFlagsAndEnv()

	os.Setenv("STORE_INTERVAL", "notanint")

	cfg, err := NewConfig()
	assert.Nil(t, cfg)
	assert.EqualError(t, err, "invalid STORE_INTERVAL env variable")
}

func TestNewConfig_InvalidRestoreEnv(t *testing.T) {
	resetFlagsAndEnv()

	os.Setenv("RESTORE", "maybe")

	cfg, err := NewConfig()
	assert.Nil(t, cfg)
	assert.EqualError(t, err, "invalid RESTORE env value, must be true or false")
}

func TestNewConfig_UnknownArgs(t *testing.T) {
	resetFlagsAndEnv()

	// Add a bogus positional arg
	os.Args = []string{"cmd", "unknownArg"}

	cfg, err := NewConfig()
	assert.Nil(t, cfg)
	assert.EqualError(t, err, "unknown flags or arguments are provided")
}

func TestNewConfig_RestoreEnv_True(t *testing.T) {
	resetFlagsAndEnv()
	os.Setenv("RESTORE", "true")

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.True(t, cfg.Restore)
}

func TestNewConfig_RestoreEnv_True_Uppercase(t *testing.T) {
	resetFlagsAndEnv()
	os.Setenv("RESTORE", "TRUE")

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.True(t, cfg.Restore)
}

func TestNewConfig_RestoreEnv_False(t *testing.T) {
	resetFlagsAndEnv()
	os.Setenv("RESTORE", "false")

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.False(t, cfg.Restore)
}

func TestNewConfig_RestoreEnv_False_MixedCase(t *testing.T) {
	resetFlagsAndEnv()
	os.Setenv("RESTORE", "False")

	cfg, err := NewConfig()
	assert.NoError(t, err)
	assert.False(t, cfg.Restore)
}
