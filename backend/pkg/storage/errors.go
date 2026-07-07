package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// translatePgError maps well-known Postgres constraint violations to the
// domain sentinels callers switch on. If the error isn't a pgconn.PgError
// or doesn't match the constraint, it's returned wrapped for diagnosis.
//
// We rely on pgconn.PgError.Code rather than parsing message text — code is
// part of the Postgres wire protocol and stable across locale versions of
// the server error message.
//
// Reference codes:
//   - 23505 = unique_violation
//   - 23503 = foreign_key_violation
//   - 23502 = not_null_violation
//   - 23514 = check_violation
func translatePgError(err error, constraintHint string, target error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			// Only translate when the violated constraint looks related to
			// the hint; otherwise let the raw error surface. This avoids
			// masking e.g. a refresh_tokens.token_hash collision as
			// ErrUserExists.
			if constraintHint == "" || strings.Contains(strings.ToLower(pgErr.ConstraintName), constraintHint) ||
				strings.Contains(strings.ToLower(pgErr.Message), constraintHint) {
				return target
			}
		}
		return fmt.Errorf("storage: pg error [%s]: %w", pgErr.Code, err)
	}
	return err
}
