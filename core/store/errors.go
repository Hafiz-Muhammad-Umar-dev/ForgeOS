package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Sentinel errors returned by Store implementations.
var (
	// ErrNoRows is returned when a query returns no rows.
	ErrNoRows = errors.New("store: no rows")

	// ErrConflict is returned when a unique constraint is violated.
	ErrConflict = errors.New("store: conflict")

	// ErrConnectionFailed is returned when the database connection fails.
	ErrConnectionFailed = errors.New("store: connection failed")

	// ErrMigrationFailed is returned when a migration fails.
	ErrMigrationFailed = errors.New("store: migration failed")

	// ErrTxFailed is returned when a transaction operation fails.
	ErrTxFailed = errors.New("store: transaction failed")
)

// MapPGError converts a pgx error to a sentinel store error.
// Returns the original error if it does not match a known pgx error.
func MapPGError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoRows
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return ErrConflict
		case "23503": // foreign_key_violation
			return ErrConflict
		case "40001": // serialization_failure
			return ErrTxFailed
		case "40P01": // deadlock_detected
			return ErrTxFailed
		}
	}
	return err
}
