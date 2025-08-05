package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestNewDB(t *testing.T) {
	dsn := ":memory:"
	driver := "sqlite"

	conn, err := New(driver, dsn)
	require.NoError(t, err)
	require.NotNil(t, conn)

	err = conn.Ping()
	assert.NoError(t, err)
}

func TestWithMaxOpenConns(t *testing.T) {
	dsn := ":memory:"
	driver := "sqlite"

	conn, err := New(driver, dsn, WithMaxOpenConns(7))
	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestWithMaxIdleConns(t *testing.T) {
	dsn := ":memory:"
	driver := "sqlite"

	conn, err := New(driver, dsn, WithMaxIdleConns(4))
	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestWithConnMaxLifetime(t *testing.T) {
	dsn := ":memory:"
	driver := "sqlite"

	conn, err := New(driver, dsn, WithConnMaxLifetime(30*time.Second))
	require.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestMultipleOptions(t *testing.T) {
	dsn := ":memory:"
	driver := "sqlite"

	conn, err := New(driver, dsn,
		WithMaxOpenConns(20),
		WithMaxIdleConns(5),
		WithConnMaxLifetime(1*time.Minute),
	)
	require.NoError(t, err)
	assert.NotNil(t, conn)
}
