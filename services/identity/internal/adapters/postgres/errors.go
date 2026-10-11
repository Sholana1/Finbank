package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	uniqueViolation        = "23505"
	invalidTextForDataType = "22P02"
)

// isInvalidUUID reports whether Postgres rejected a value because it is
// not a well-formed uuid. A caller that sends "abc" as a session id
// should be told "not found", not shown an internal error.
func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == invalidTextForDataType
}
