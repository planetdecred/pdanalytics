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

	lastMempoolBlockHeight = `SELECT last_block_height FROM mempool ORDER BY last_block_height DESC LIMIT 1`
	lastMempoolEntryTime   = `SELECT time FROM mempool ORDER BY time DESC LIMIT 1`

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

func (db *PgDb) CreateTables(ctx context.Context) error {
	if !db.mempoolDataTableExits() {
		if err := db.createMempoolDataTable(); err != nil {
			return err
		}
	}
	if !db.mempoolBinDataTableExits() {
		if err := db.createMempoolDayBinTable(); err != nil {
			return err
		}
	}
	if !db.NetworkNodeTableExists() {
		if err := db.CreateNetworkNodeTable(); err != nil {
			return err
		}
	}
	if !db.NetworkSnapshotBinTableExists() {
		if err := db.CreateNetworkSnapshotBinTable(); err != nil {
			return err
		}
	}
	if !db.NodeVersionTableExists() {
		if err := db.CreateNodeVersoinTable(); err != nil {
			return err
		}
	}
	if !db.NodeLocationTableExists() {
		if err := db.CreateNodeLocationTable(); err != nil {
			return err
		}
	}
	if !db.NetworkNodeTableExists() {
		if err := db.CreateNetworkNodeTable(); err != nil {
			return err
		}
	}
	if !db.HeartbeatTableExists() {
		if err := db.CreateHeartbeatTable(); err != nil {
			return err
		}
	}
	if !db.propagationTableExists() {
		if err := db.createPropagationTable(); err != nil {
			return err
		}
	}
	if !db.BlockTableExits() {
		if err := db.CreateBlockTable(); err != nil {
			return err
		}
	}
	if !db.blockBinTableExits() {
		if err := db.createBlockBinTable(); err != nil {
			return err
		}
	}
	if !db.VoteTableExits() {
		if err := db.CreateVoteTable(); err != nil {
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

// Mempool tables
func (pg *PgDb) createMempoolDataTable() error {
	_, err := pg.db.Exec(createMempoolTable)
	return err
}

func (pg *PgDb) createMempoolDayBinTable() error {
	_, err := pg.db.Exec(createMempoolDayBinTable)
	return err
}

func (pg *PgDb) mempoolDataTableExits() bool {
	exists, _ := pg.tableExists("mempool")
	return exists
}

func (pg *PgDb) mempoolBinDataTableExits() bool {
	exists, _ := pg.tableExists("mempool_bin")
	return exists
}

// network snapshot
func (pg *PgDb) CreateNetworkSnapshotTable() error {
	_, err := pg.db.Exec(createNetworkSnapshotTable)
	return err
}

func (pg *PgDb) NetworkSnapshotTableExists() bool {
	exists, _ := pg.tableExists("network_snapshot")
	return exists
}

// network_snapshot_bin
func (pg *PgDb) CreateNetworkSnapshotBinTable() error {
	_, err := pg.db.Exec(createNetworkSnapshotBinTable)
	return err
}

func (pg *PgDb) NetworkSnapshotBinTableExists() bool {
	exists, _ := pg.tableExists("network_snapshot_bin")
	return exists
}

// node_version
func (pg *PgDb) CreateNodeVersoinTable() error {
	_, err := pg.db.Exec(createNodeVersionTable)
	return err
}

func (pg *PgDb) NodeVersionTableExists() bool {
	exists, _ := pg.tableExists("node_version")
	return exists
}

// node_location
func (pg *PgDb) CreateNodeLocationTable() error {
	_, err := pg.db.Exec(createNodeLocationTable)
	return err
}

func (pg *PgDb) NodeLocationTableExists() bool {
	exists, _ := pg.tableExists("node_location")
	return exists
}

// network node
func (pg *PgDb) CreateNetworkNodeTable() error {
	_, err := pg.db.Exec(createNodeTable)
	return err
}

func (pg *PgDb) NetworkNodeTableExists() bool {
	exists, _ := pg.tableExists("node")
	return exists
}

// network peer
func (pg *PgDb) CreateHeartbeatTable() error {
	_, err := pg.db.Exec(createHeartbeatTable)
	return err
}

func (pg *PgDb) HeartbeatTableExists() bool {
	exists, _ := pg.tableExists("heartbeat")
	return exists
}

func (pg *PgDb) createPropagationTable() error {
	_, err := pg.db.Exec(createPropagationTableScript)
	return err
}

func (pg *PgDb) propagationTableExists() bool {
	exists, _ := pg.tableExists("propagation")
	return exists
}

// block table
func (pg *PgDb) CreateBlockTable() error {
	_, err := pg.db.Exec(createBlockTableScript)
	return err
}

func (pg *PgDb) BlockTableExits() bool {
	exists, _ := pg.tableExists("block")
	return exists
}

// createBlockBinTable
func (pg *PgDb) createBlockBinTable() error {
	_, err := pg.db.Exec(createBlockBinTableScript)
	return err
}

func (pg *PgDb) blockBinTableExits() bool {
	exists, _ := pg.tableExists("block_bin")
	return exists
}

// vote table
func (pg *PgDb) CreateVoteTable() error {
	_, err := pg.db.Exec(createVoteTableScript)
	return err
}

func (pg *PgDb) VoteTableExits() bool {
	exists, _ := pg.tableExists("vote")
	return exists
}

// vote_receive_time_deviation table
func (pg *PgDb) createVoteReceiveTimeDeviationTable() error {
	_, err := pg.db.Exec(createVoteReceiveTimeDeviationTableScript)
	return err
}

func (pg *PgDb) voteReceiveTimeDeviationTableExits() bool {
	exists, _ := pg.tableExists("vote_receive_time_deviation")
	return exists
}

func (pg *PgDb) tableExists(name string) (bool, error) {
	rows, err := pg.db.Query(`SELECT relname FROM pg_class WHERE relname = $1`, name)
	if err == nil {
		defer func() {
			if e := rows.Close(); e != nil {
				log.Error("Close of Query failed: ", e)
			}
		}()
		return rows.Next(), nil
	}
	return false, err
}

func (pg *PgDb) DropTables() error {

	if err := pg.dropTable("mempool"); err != nil {
		return err
	}

	if err := pg.dropTable("mempool_bin"); err != nil {
		return err
	}

	if err := pg.dropTable("network_snapshot"); err != nil {
		return err
	}

	if err := pg.dropTable("network_snapshot_bin"); err != nil {
		return err
	}

	if err := pg.dropTable("heartbeat"); err != nil {
		return err
	}

	if err := pg.dropTable("node"); err != nil {
		return err
	}

	// propagation
	if err := pg.dropTable("propagation"); err != nil {
		return err
	}

	// block
	if err := pg.dropTable("block"); err != nil {
		return err
	}

	// block_bin
	if err := pg.dropTable("block_bin"); err != nil {
		return err
	}

	// vote
	if err := pg.dropTable("vote"); err != nil {
		return err
	}

	// vote_receive_time_deviation
	if err := pg.dropTable("vote_receive_time_deviation"); err != nil {
		return err
	}
	return nil
}

func (pg *PgDb) DropCacheTables() error {
	if err := pg.dropTable("network_snapshot_bin"); err != nil {
		return err
	}

	if err := pg.dropTable("mempool_bin"); err != nil {
		return err
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
