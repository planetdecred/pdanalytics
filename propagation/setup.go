package propagation

//go:generate sqlboiler --wipe psql --no-hooks --no-auto-timestamps

import (
	"context"

	"github.com/planetdecred/pdanalytics/dbhelpers"
)

const (
	createPropagationTableScript = `CREATE TABLE IF NOT EXISTS propagation (
		height INT8 NOT NULL,
		time INT8 NOT NULL,
		bin VARCHAR(25) NOT NULL,
		source VARCHAR(255) NOT NULL,
		deviation FLOAT8 NOT NULL,
		PRIMARY KEY (height, source, bin)
	);`

	lastMempoolBlockHeight = `SELECT last_block_height FROM mempool ORDER BY last_block_height DESC LIMIT 1`
	lastMempoolEntryTime   = `SELECT time FROM mempool ORDER BY time DESC LIMIT 1`

	createBlockTableScript = `CREATE TABLE IF NOT EXISTS block (
		height INT,
		receive_time timestamp,
		internal_timestamp timestamp,
		hash VARCHAR(512),
		PRIMARY KEY (height)
	);`

	createBlockBinTableScript = `CREATE TABLE IF NOT EXISTS block_bin (
		height INT8 NOT NULL,
		receive_time_diff FLOAT8 NOT NULL,
		internal_timestamp INT8 NOT NULL,
		bin VARCHAR(25) NOT NULL,
		PRIMARY KEY (height,bin)
	);`

	createVoteTableScript = `CREATE TABLE IF NOT EXISTS vote (
		hash VARCHAR(128),
		voting_on INT8,
		block_hash VARCHAR(128),
		receive_time timestamp,
		block_receive_time timestamp,
		targeted_block_time timestamp,
		validator_id INT,
		validity VARCHAR(128),
		PRIMARY KEY (hash)
	);`

	createVoteReceiveTimeDeviationTableScript = `CREATE TABLE IF NOT EXISTS vote_receive_time_deviation (
		bin VARCHAR(25) NOT NULL,
		block_height INT8 NOT NULL,
		block_time INT8 NOT NULL,
		receive_time_difference FLOAT8 NOT NULL,
		PRIMARY KEY (block_time,bin)
	);`
)

func (db *PgDb) CreateTables(ctx context.Context) error {
	if !db.propagationTableExists() {
		if err := db.createPropagationTable(); err != nil {
			return err
		}
	}
	if !db.blockTableExits() {
		if err := db.createBlockTable(); err != nil {
			return err
		}
	}
	if !db.blockBinTableExits() {
		if err := db.createBlockBinTable(); err != nil {
			return err
		}
	}
	if !db.voteTableExits() {
		if err := db.createVoteTable(); err != nil {
			return err
		}
	}
	if !db.voteReceiveTimeDeviationTableExits() {
		if err := db.createVoteReceiveTimeDeviationTable(); err != nil {
			return err
		}
	}
	return nil
}

func (pg *PgDb) createPropagationTable() error {
	_, err := pg.db.Exec(createPropagationTableScript)
	return err
}

func (pg *PgDb) propagationTableExists() bool {
	exists, _ := dbhelpers.TableExists(pg.db, "propagation")
	return exists
}

// block table
func (pg *PgDb) createBlockTable() error {
	_, err := pg.db.Exec(createBlockTableScript)
	return err
}

func (pg *PgDb) blockTableExits() bool {
	exists, _ := dbhelpers.TableExists(pg.db, "block")
	return exists
}

// createBlockBinTable
func (pg *PgDb) createBlockBinTable() error {
	_, err := pg.db.Exec(createBlockBinTableScript)
	return err
}

func (pg *PgDb) blockBinTableExits() bool {
	exists, _ := dbhelpers.TableExists(pg.db, "block_bin")
	return exists
}

// vote table
func (pg *PgDb) createVoteTable() error {
	_, err := pg.db.Exec(createVoteTableScript)
	return err
}

func (pg *PgDb) voteTableExits() bool {
	exists, _ := dbhelpers.TableExists(pg.db, "vote")
	return exists
}

// vote_receive_time_deviation table
func (pg *PgDb) createVoteReceiveTimeDeviationTable() error {
	_, err := pg.db.Exec(createVoteReceiveTimeDeviationTableScript)
	return err
}

func (pg *PgDb) voteReceiveTimeDeviationTableExits() bool {
	exists, _ := dbhelpers.TableExists(pg.db, "vote_receive_time_deviation")
	return exists
}

func (pg *PgDb) DropTables() error {

	// propagation
	if err := dbhelpers.DropTable(pg.db, "propagation"); err != nil {
		return err
	}

	// block
	if err := dbhelpers.DropTable(pg.db, "block"); err != nil {
		return err
	}

	// block_bin
	if err := dbhelpers.DropTable(pg.db, "block_bin"); err != nil {
		return err
	}

	// vote
	if err := dbhelpers.DropTable(pg.db, "vote"); err != nil {
		return err
	}

	// vote_receive_time_deviation
	if err := dbhelpers.DropTable(pg.db, "vote_receive_time_deviation"); err != nil {
		return err
	}

	return nil
}
