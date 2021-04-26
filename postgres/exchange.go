package postgres

var (
	createExchangeTable = `CREATE TABLE IF NOT EXISTS exchange (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL);`

	createExchangeTickTable = `CREATE TABLE IF NOT EXISTS exchange_tick (
		id SERIAL PRIMARY KEY,
		exchange_id INT REFERENCES exchange(id) NOT NULL, 
		interval INT NOT NULL,
		high FLOAT NOT NULL,
		low FLOAT NOT NULL,
		open FLOAT NOT NULL,
		close FLOAT NOT NULL,
		volume FLOAT NOT NULL,
		currency_pair TEXT NOT NULL,
		time TIMESTAMPTZ NOT NULL
	);`

	createExchangeTickIndex = `CREATE UNIQUE INDEX IF NOT EXISTS exchange_tick_idx ON exchange_tick (exchange_id, interval, currency_pair, time);`

	lastExchangeTickEntryTime = `SELECT time FROM exchange_tick ORDER BY time DESC LIMIT 1`

	lastExchangeEntryID = `SELECT id FROM exchange ORDER BY id DESC LIMIT 1`
)
