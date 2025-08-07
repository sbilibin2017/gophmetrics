package server

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

var (
	addr            string
	storeInterval   int
	fileStoragePath string
	restore         bool
	databaseDSN     string
	migrationsDir   string = "migrations"
	key             string // <--- добавляем переменную для ключа
)

func init() {
	pflag.StringVarP(&addr, "address", "a", "localhost:8080", "server URL")
	pflag.IntVarP(&storeInterval, "interval", "i", 300, "interval in seconds to save metrics (0 = sync save)")
	pflag.StringVarP(&fileStoragePath, "file", "f", "metrics.json", "file path to store metrics")
	pflag.BoolVarP(&restore, "restore", "r", true, "restore metrics from file on startup")
	pflag.StringVarP(&databaseDSN, "database-dsn", "d", "", "PostgreSQL DSN connection string")

	pflag.StringVarP(&key, "key", "k", "", "secret key for SHA256 hash")
}

// Config holds all server configuration values
type Config struct {
	Addr            string `json:"address"`
	StoreInterval   int    `json:"store_interval"`
	FileStoragePath string `json:"file_storage_path"`
	Restore         bool   `json:"restore"`
	DatabaseDSN     string `json:"database_dsn"`
	MigrationsDir   string `json:"migrations_dir"`
	Key             string `json:"key"`
}

// NewConfig parses flags, environment variables, and returns a Config struct or an error
func NewConfig() (*Config, error) {
	pflag.Parse()

	if len(pflag.Args()) > 0 {
		return nil, errors.New("unknown flags or arguments are provided")
	}

	if env := os.Getenv("ADDRESS"); env != "" {
		addr = env
	}

	if env := os.Getenv("STORE_INTERVAL"); env != "" {
		i, err := strconv.Atoi(env)
		if err != nil {
			return nil, errors.New("invalid STORE_INTERVAL env variable")
		}
		storeInterval = i
	}

	if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
		fileStoragePath = env
	}

	if env := os.Getenv("RESTORE"); env != "" {
		switch strings.ToLower(env) {
		case "true":
			restore = true
		case "false":
			restore = false
		default:
			return nil, errors.New("invalid RESTORE env value, must be true or false")
		}
	}

	if env := os.Getenv("DATABASE_DSN"); env != "" {
		databaseDSN = env
	}

	if key == "" {
		key = os.Getenv("KEY")
	}

	cfg := &Config{
		Addr:            addr,
		StoreInterval:   storeInterval,
		FileStoragePath: fileStoragePath,
		Restore:         restore,
		DatabaseDSN:     databaseDSN,
		MigrationsDir:   migrationsDir,
		Key:             key,
	}

	return cfg, nil
}
