package store

import "time"

// Config configures the PostgreSQL store adapter.
type Config struct {
	// DSN is the PostgreSQL connection string (e.g., "postgres://user:pass@localhost:5432/devos").
	DSN string

	// MaxOpenConns is the maximum number of open connections to the database.
	// Zero means unlimited.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections in the pool.
	// Zero means no idle connections are retained.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	// Zero means unlimited.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	// Zero means unlimited.
	ConnMaxIdleTime time.Duration

	// ConnectTimeout is the maximum time to wait for connection establishment.
	ConnectTimeout time.Duration

	// EnableSSL controls whether to require SSL connections.
	EnableSSL bool
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() Config {
	return Config{
		DSN:             "postgres://localhost:5432/devos?sslmode=disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnectTimeout:  10 * time.Second,
		EnableSSL:       false,
	}
}
