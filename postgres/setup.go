package postgres

import (
	"context"
	"fmt"
)

const (
	createMempoolTable = `CREATE TABLE IF NOT EXISTS mempool (
		time timestamp,
		first_seen_time timestamp,
		number_of_transactions INT,
		voters INT,
		tickets INT,
		revocations INT,
		size INT,
		total_fee FLOAT8,
		total FLOAT8,
		PRIMARY KEY (time)
	);`

	createMempoolDayBinTable = `CREATE TABLE IF NOT EXISTS mempool_bin (
		time INT8,
		bin VARCHAR(25),
		number_of_transactions INT,
		size INT,
		total_fee FLOAT8,
		PRIMARY KEY (time,bin)
	);`

	createNetworkSnapshotTable = `CREATE TABLE If NOT EXISTS network_snapshot (
		timestamp INT8 NOT NULL,
		height INT8 NOT NULL,
		node_count INT NOT NULL,
		reachable_nodes INT NOT NULL,
		oldest_node VARCHAR(256) NOT NULL DEFAULT '',
		oldest_node_timestamp INT8 NOT NULL DEFAULT 0,
		latency INT NOT NULL DEFAULT 0,
		PRIMARY KEY (timestamp)
	);`

	createNetworkSnapshotBinTable = `CREATE TABLE If NOT EXISTS network_snapshot_bin (
		timestamp INT8 NOT NULL,
		height INT8 NOT NULL,
		node_count INT NOT NULL,
		reachable_nodes INT NOT NULL,
		bin VARCHAR(25) NOT NULL DEFAULT '',
		PRIMARY KEY (timestamp, bin)
	);`

	createNodeVersionTable = `CREATE TABLE If NOT EXISTS node_version (
		timestamp INT8 NOT NULL,
		height INT8 NOT NULL,
		node_count INT NOT NULL,
		user_agent VARCHAR(256) NOT NULL,
		bin VARCHAR(25) NOT NULL DEFAULT '',
		PRIMARY KEY (timestamp, bin, user_agent)
	);`

	createNodeLocationTable = `CREATE TABLE If NOT EXISTS node_location (
		timestamp INT8 NOT NULL,
		height INT8 NOT NULL,
		node_count INT NOT NULL,
		country VARCHAR(256) NOT NULL,
		bin VARCHAR(25) NOT NULL DEFAULT '',
		PRIMARY KEY (timestamp, bin, country)
	);`

	createNodeTable = `CREATE TABLE If NOT EXISTS node (
		address VARCHAR(256) NOT NULL PRIMARY KEY,
		ip_version INT NOT NULL,
		country VARCHAR(256) NOT NULL,
		region VARCHAR(256) NOT NULL,
		city VARCHAR(256) NOT NULL,
		zip VARCHAR(256) NOT NULL,
		last_attempt INT8 NOT NULL,
		last_seen INT8 NOT NULL,
		last_success INT8 NOT NULL,
		failure_count INT NOT NULL DEFAULT 0,
		is_dead BOOLEAN NOT NULL,
		connection_time INT8 NOT NULL,
		protocol_version INT NOT NULL,
		user_agent VARCHAR(256) NOT NULL,
		services VARCHAR(256) NOT NULL,
		starting_height INT8 NOT NULL,
		current_height INT8 NOT NULL
	);`

	createHeartbeatTable = `CREATE TABLE If NOT EXISTS heartbeat (
		timestamp INT8 NOT NULL,
		node_id VARCHAR(256) NOT NULL REFERENCES node(address),
		last_seen INT8 NOT NULL,
		latency INT NOT NULL,
		current_height INT8 NOT NULL,
		PRIMARY KEY (timestamp, node_id)
	);`

	createPropagationTableScript = `CREATE TABLE IF NOT EXISTS propagation (
		height INT8 NOT NULL,
		time INT8 NOT NULL,
		bin VARCHAR(25) NOT NULL,
		source VARCHAR(255) NOT NULL,
		deviation FLOAT8 NOT NULL,
		PRIMARY KEY (height, source, bin)
	);`

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

var (
	createTableScripts = map[string]string{
		"mempool":                     createMempoolTable,
		"mempool_bin":                 createMempoolDayBinTable,
		"network_snapshot":            createNetworkSnapshotTable,
		"network_snapshot_bin":        createNetworkSnapshotBinTable,
		"node_version":                createNodeVersionTable,
		"node_location":               createNodeLocationTable,
		"node":                        createNodeTable,
		"heartbeat":                   createHeartbeatTable,
		"propagation":                 createPropagationTableScript,
		"block":                       createBlockTableScript,
		"block_bin":                   createBlockBinTableScript,
		"vote":                        createVoteTableScript,
		"vote_receive_time_deviation": createVoteReceiveTimeDeviationTableScript,
		"proposals":                   createProposalTableScript,
		"proposal_votes":              createProposalVotesTableScript,
		"agendas":                     CreateAgendasTable,
		"agenda_votes":                CreateAgendaVotesTable,
	}

	tableOrder = []string{
		"mempool",
		"mempool_bin",
		"network_snapshot",
		"network_snapshot_bin",
		"node_version",
		"node_location",
		"node",
		"heartbeat",
		"propagation",
		"block",
		"block_bin",
		"vote",
		"vote_receive_time_deviation",
		"proposals",
		"proposal_votes",
		"agendas",
		"agenda_votes",
	}

	// createIndexScripts is a map of table name to a collection of index on the table
	createIndexScripts = map[string][]string{
		"proposals": {
			IndexProposalsTableOnToken,
		},
		"proposal_votes": {
			IndexProposalVotesTableOnProposalsID,
		},
		"agendas": {
			IndexAgendasTableOnAgendaID,
		},
		"agenda_votes": {
			IndexAgendaVotesTableOnAgendaID,
		},
	}
)

func (pg *PgDb) CreateTables(ctx context.Context) error {
	tx, err := pg.db.Begin()
	if err != nil {
		return err
	}
	for _, tableName := range tableOrder {
		if exist := pg.TableExists(tableName); exist {
			continue
		}
		_, err := tx.Exec(createTableScripts[tableName])
		if err != nil {
			_ = tx.Rollback()
			log.Errorf("an error occured while running %s", createTableScripts[tableName])
			return err
		}
		for _, createScript := range createIndexScripts[tableName] {
			_, err := tx.Exec(createScript)
			if err != nil {
				_ = tx.Rollback()
				log.Errorf("an error occured while running %s", createIndexScripts[tableName])
				return err
			}
		}

		return tx.Commit()
	}
	return nil
}

func (pg *PgDb) TableExists(name string) bool {
	rows, err := pg.db.Query(`SELECT relname FROM pg_class WHERE relname = $1`, name)
	if err == nil {
		defer func() {
			if e := rows.Close(); e != nil {
				log.Error("Close of Query failed: ", e)
			}
		}()
		return rows.Next()
	}
	return false
}

func (pg *PgDb) DropTables() error {
	for tableName := range createTableScripts {
		if err := pg.dropTable(tableName); err != nil {
			return err
		}
	}
	return nil
}

func (pg *PgDb) dropTable(name string) error {
	log.Tracef("Dropping table %s", name)
	_, err := pg.db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s;`, name))
	return err
}

func (pg *PgDb) dropIndex(name string) error {
	log.Tracef("Dropping table %s", name)
	_, err := pg.db.Exec(fmt.Sprintf(`DROP INDEX IF EXISTS %s;`, name))
	return err
}
