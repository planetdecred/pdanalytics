package postgres

import "fmt"

const (
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
)

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

func (pg *PgDb) DropAllTables() error {

	// network_snapshot
	if err := pg.dropTable("network_snapshot"); err != nil {
		return err
	}

	//network_snapshot_bin
	if err := pg.dropTable("network_snapshot_bin"); err != nil {
		return err
	}

	// heartbeat
	if err := pg.dropTable("heartbeat"); err != nil {
		return err
	}

	// node
	if err := pg.dropTable("node"); err != nil {
		return err
	}

	return nil
}

func (pg *PgDb) DropCacheTables() error {
	//network_snapshot_bin
	if err := pg.dropTable("network_snapshot_bin"); err != nil {
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
