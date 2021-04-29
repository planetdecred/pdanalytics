package postgres

const (
	createVSPInfoTable = `CREATE TABLE IF NOT EXISTS vsp (
		id SERIAL PRIMARY KEY,
		name TEXT,
		api_enabled BOOLEAN,
		api_versions_supported INT8[],
		network TEXT,
		url TEXT,
		launched TIMESTAMPTZ
	);`

	createVSPTickTable = `CREATE TABLE IF NOT EXISTS vsp_tick (
		id SERIAL PRIMARY KEY,
		vsp_id INT REFERENCES vsp(id) NOT NULL,
		immature INT NOT NULL,
		live INT NOT NULL,
		voted INT NOT NULL,
		missed INT NOT NULL,
		pool_fees FLOAT NOT NULL,
		proportion_live FLOAT NOT NULL,
		proportion_missed FLOAT NOT NULL,
		user_count INT NOT NULL,
		users_active INT NOT NULL,
		time TIMESTAMPTZ NOT NULL
	);`

	createVSPTickBinTable = `CREATE TABLE IF NOT EXISTS vsp_tick_bin (
		vsp_id INT REFERENCES vsp(id) NOT NULL,
		bin VARCHAR(25), 
		immature INT,
		live INT,
		voted INT,
		missed INT,
		pool_fees FLOAT,
		proportion_live FLOAT,
		proportion_missed FLOAT,
		user_count INT,
		users_active INT,
		time INT8,
		PRIMARY KEY (vsp_id, time, bin)
	);`

	createVSPTickIndex = `CREATE UNIQUE INDEX IF NOT EXISTS vsp_tick_idx ON vsp_tick (vsp_id,immature,live,voted,missed,pool_fees,proportion_live,proportion_missed,user_count,users_active, time);`

	lastVspTickEntryTime = `SELECT time FROM vsp_tick ORDER BY time DESC LIMIT 1`
)
