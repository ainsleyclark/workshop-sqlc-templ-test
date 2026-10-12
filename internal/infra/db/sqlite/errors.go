package sqlite

import (
	"errors"

	"modernc.org/sqlite"
	"modernc.org/sqlite/lib"
)

// IsUniqueViolation reports whether err is SQLite rejecting a
// duplicate value, in a UNIQUE column or the primary key. SQLite gives
// the two their own codes, but either way the row already exists.
func IsUniqueViolation(err error) bool {
	sqliteErr, ok := errors.AsType[*sqlite.Error](err)
	if !ok {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}
