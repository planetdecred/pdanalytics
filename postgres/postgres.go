// Copyright (c) 2018-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package postgres

//go:generate sqlboiler --wipe psql --no-hooks --no-auto-timestamps

import (
	"database/sql"
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
