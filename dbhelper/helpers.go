package dbhelper

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

var (
	// PGCancelError is the error string PostgreSQL returns when a query fails
	// to complete due to user requested cancellation.
	PGCancelError       = "pq: canceling statement due to user request"
	CtxDeadlineExceeded = context.DeadlineExceeded.Error()
	TimeoutPrefix       = "TIMEOUT of PostgreSQL query"
)

const DateTemplate = "2006-01-02 15:04"
const DateMiliTemplate = "2006-01-02 15:04:05.99"

func TableExists(db *sql.DB, name string) (bool, error) {
	rows, err := db.Query(`SELECT relname FROM pg_class WHERE relname = $1`, name)
	if err == nil {
		defer func() {
			if e := rows.Close(); e != nil {
				fmt.Println("Close of Query failed: ", e)
			}
		}()
		return rows.Next(), nil
	}
	return false, err
}

func DropTable(db *sql.DB, name string) error {
	fmt.Println("Dropping table ", name)
	_, err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s;`, name))
	return err
}

// IsTimeout checks if the message is prefixed with the expected DB timeout
// message prefix.
func IsTimeout(msg string) bool {
	// Contains is used instead of HasPrefix since error messages are often
	// supplemented with additional information.
	return strings.Contains(msg, TimeoutPrefix) ||
		strings.Contains(msg, CtxDeadlineExceeded)
}

// IsTimeoutErr checks if error's message is prefixed with the expected DB
// timeout message prefix.
func IsTimeoutErr(err error) bool {
	return err != nil && IsTimeout(err.Error())
}
