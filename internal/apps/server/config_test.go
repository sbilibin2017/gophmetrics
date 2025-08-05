package server

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func resetFlags() {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
}

func TestNewConfigFromFlags_Defaults(t *testing.T) {
	// Reset flags and os.Args before test
	resetFlags()
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}

	cfg, err := NewConfigFromFlags()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "localhost:8080", cfg.Address)
	require.Equal(t, 300, cfg.StoreInterval)
	require.Equal(t, "metrics.json", cfg.FileStoragePath)
	require.Equal(t, true, cfg.Restore)
}

func TestNewConfigFromFlags_CustomFlags(t *testing.T) {
	resetFlags()
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{
		"cmd",
		"--address=127.0.0.1:9090",
		"--interval=10",
		"--file=myfile.json",
		"--restore=false",
	}

	cfg, err := NewConfigFromFlags()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "127.0.0.1:9090", cfg.Address)
	require.Equal(t, 10, cfg.StoreInterval)
	require.Equal(t, "myfile.json", cfg.FileStoragePath)
	require.Equal(t, false, cfg.Restore)
}

func TestNewConfigFromFlags_UnknownArgs(t *testing.T) {
	resetFlags()
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{
		"cmd",
		"--address=127.0.0.1:9090",
		"unexpectedArg",
	}

	_, err := NewConfigFromFlags()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown flags or arguments are provided")
}

func TestNewConfigFromEnv_AllSet(t *testing.T) {
	os.Setenv("ADDRESS", "192.168.1.1:8080")
	os.Setenv("STORE_INTERVAL", "20")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/metrics.json")
	os.Setenv("RESTORE", "false")
	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("STORE_INTERVAL")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("RESTORE")
	}()

	cfg, err := NewConfigFromEnv()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "192.168.1.1:8080", cfg.Address)
	require.Equal(t, 20, cfg.StoreInterval)
	require.Equal(t, "/tmp/metrics.json", cfg.FileStoragePath)
	require.Equal(t, false, cfg.Restore)
}

func TestNewConfigFromEnv_PartialEnv(t *testing.T) {
	os.Setenv("ADDRESS", "10.0.0.1:9090")
	defer os.Unsetenv("ADDRESS")

	cfg, err := NewConfigFromEnv()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "10.0.0.1:9090", cfg.Address)
	// Other fields are defaults, so you can check accordingly if defaults are applied in configs.NewServerConfig
}

func TestNewConfigFromEnv_InvalidStoreInterval(t *testing.T) {
	os.Setenv("STORE_INTERVAL", "notanint")
	defer os.Unsetenv("STORE_INTERVAL")

	_, err := NewConfigFromEnv()
	require.Error(t, err)
}

func TestNewConfigFromEnv_InvalidRestoreValue(t *testing.T) {
	os.Setenv("RESTORE", "maybe")
	defer os.Unsetenv("RESTORE")

	_, err := NewConfigFromEnv()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid RESTORE value")
}

func TestNewConfigFromEnv_RestoreTrueCaseInsensitive(t *testing.T) {
	os.Setenv("RESTORE", "TrUe")
	defer os.Unsetenv("RESTORE")

	cfg, err := NewConfigFromEnv()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, true, cfg.Restore)
}
