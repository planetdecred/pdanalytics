// Copyright (c) 2018-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package postgres

//go:generate sqlboiler --wipe psql --no-hooks --no-auto-timestamps

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/planetdecred/pdanalytics/dbhelper"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type PgDb struct {
	db           *sql.DB
	queryTimeout time.Duration
}

type logWriter struct{}

func (l logWriter) Write(p []byte) (n int, err error) {
	log.Debug(string(p))
	return len(p), nil
}

func NewPgDb(host, port, user, pass, dbname string, debug bool) (*PgDb, error) {
	db, err := dbhelper.Connect(host, port, user, pass, dbname)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	if debug {
		boil.DebugMode = true
		boil.DebugWriter = logWriter{}
	}
	return &PgDb{
		db:           db,
		queryTimeout: time.Second * 30,
	}, nil
}

func (pg *PgDb) Close() error {
	log.Trace("Closing postgresql connection")
	return pg.db.Close()
}

func (pg *PgDb) timeoutError() string {
	return fmt.Sprintf("%s after %v", dbhelper.TimeoutPrefix, pg.queryTimeout)
}

// replaceCancelError will replace the generic error strings that can occur when
// a PG query is canceled (dbtypes.PGCancelError) or a context deadline is
// exceeded (dbtypes.CtxDeadlineExceeded from context.DeadlineExceeded).
func (pg *PgDb) replaceCancelError(err error) error {
	if err == nil {
		return err
	}

	patched := err.Error()
	if strings.Contains(patched, dbhelper.PGCancelError) {
		patched = strings.Replace(patched, dbhelper.PGCancelError,
			pg.timeoutError(), -1)
	} else if strings.Contains(patched, dbhelper.CtxDeadlineExceeded) {
		patched = strings.Replace(patched, dbhelper.CtxDeadlineExceeded,
			pg.timeoutError(), -1)
	} else {
		return err
	}
	return errors.New(patched)
}
