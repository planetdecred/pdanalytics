package dbhelpers

import (
	"database/sql"
	"fmt"
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
