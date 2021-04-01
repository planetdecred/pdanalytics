package postgres

import "github.com/planetdecred/pdanalytics/dbhelper"

const (
	// agendas table

	CreateAgendasTable = `CREATE TABLE IF NOT EXISTS agendas (
		id SERIAL PRIMARY KEY,
		name TEXT,
		status INT2,
		locked_in INT4,
		activated INT4,
		hard_forked INT4
	);`

	// index
	IndexOfAgendasTableOnName = "uix_agendas_name"

	// Insert
	insertAgendaRow = `INSERT INTO agendas (name, status, locked_in, activated,
	hard_forked) VALUES ($1, $2, $3, $4, $5) `

	InsertAgendaRow = insertAgendaRow + `RETURNING id;`

	UpsertAgendaRow = insertAgendaRow + `ON CONFLICT (name) DO UPDATE
	SET status = $2, locked_in = $3, activated = $4, hard_forked = $5 RETURNING id;`

	IndexAgendasTableOnAgendaID = `CREATE UNIQUE INDEX ` + IndexOfAgendasTableOnName +
		` ON agendas(name);`
	DeindexAgendasTableOnAgendaID = `DROP INDEX ` + IndexOfAgendasTableOnName + ` CASCADE;`

	SelectAllAgendas = `SELECT id, name, status, locked_in, activated, hard_forked
	FROM agendas;`

	SelectAgendasLockedIn = `SELECT locked_in FROM agendas WHERE name = $1;`

	SelectAgendasHardForked = `SELECT hard_forked FROM agendas WHERE name = $1;`

	SelectAgendasActivated = `SELECT activated FROM agendas WHERE name = $1;`

	SetVoteMileStoneheights = `UPDATE agendas SET status = $2, locked_in = $3,
	activated = $4, hard_forked = $5 WHERE id = $1;`

	// DeleteAgendasDuplicateRows removes rows that would violate the unique
	// index uix_agendas_name. This should be run prior to creating the index.
	DeleteAgendasDuplicateRows = `DELETE FROM agendas
	WHERE id IN (SELECT id FROM (
			SELECT id, ROW_NUMBER()
			OVER (partition BY name ORDER BY id) AS rnum
			FROM agendas) t
		WHERE t.rnum > 1);`

	// agendas votes table

	CreateAgendaVotesTable = `CREATE TABLE IF NOT EXISTS agenda_votes (
		id SERIAL PRIMARY KEY,
		votes_row_id INT8,
		agendas_row_id INT8,
		agenda_vote_choice INT2
	);`

	// index
	IndexOfAgendaVotesTableOnRowIDs = "uix_agenda_votes"

	// Insert
	insertAgendaVotesRow = `INSERT INTO agenda_votes (votes_row_id, agendas_row_id,
	agenda_vote_choice) VALUES ($1, $2, $3) `

	InsertAgendaVotesRow = insertAgendaVotesRow + `RETURNING id;`

	UpsertAgendaVotesRow = insertAgendaVotesRow + `ON CONFLICT (agendas_row_id,
	votes_row_id) DO UPDATE SET agenda_vote_choice = $3 RETURNING id;`

	IndexAgendaVotesTableOnAgendaID = `CREATE UNIQUE INDEX ` + IndexOfAgendaVotesTableOnRowIDs +
		` ON agenda_votes(votes_row_id, agendas_row_id);`
	DeindexAgendaVotesTableOnAgendaID = `DROP INDEX ` + IndexOfAgendaVotesTableOnRowIDs + ` CASCADE;`

	// DeleteAgendaVotesDuplicateRows removes rows that would violate the unique
	// index uix_agenda_votes. This should be run prior to creating the index.
	DeleteAgendaVotesDuplicateRows = `DELETE FROM agenda_votes
	WHERE id IN (SELECT id FROM (
			SELECT id, ROW_NUMBER()
			OVER (partition BY votes_row_id, agendas_row_id ORDER BY id) AS rnum
			FROM agenda_votes) t
		WHERE t.rnum > 1);`

	// Select

	SelectAgendasVotesByTime = `SELECT votes.block_time AS timestamp,` +
		selectAgendaVotesQuery + `GROUP BY timestamp ORDER BY timestamp;`

	SelectAgendasVotesByHeight = `SELECT votes.height AS height,` +
		selectAgendaVotesQuery + `GROUP BY height ORDER BY height;`

	SelectAgendaVoteTotals = `SELECT ` + selectAgendaVotesQuery + `;`

	selectAgendaVotesQuery = `
		count(CASE WHEN agenda_votes.agenda_vote_choice = $1 THEN 1 ELSE NULL END) AS yes,
		count(CASE WHEN agenda_votes.agenda_vote_choice = $2 THEN 1 ELSE NULL END) AS abstain,
		count(CASE WHEN agenda_votes.agenda_vote_choice = $3 THEN 1 ELSE NULL END) AS no,
		count(*) AS total
	FROM agenda_votes
	INNER JOIN votes ON agenda_votes.votes_row_id = votes.id
	WHERE agenda_votes.agendas_row_id = (SELECT id from agendas WHERE name = $4)
		AND votes.height >= $5 AND votes.height <= $6
		AND votes.is_mainchain = TRUE `
)

var agendasVotesSummaries = map[string]*dbhelper.AgendaSummary{
	"treasury": {
		Yes:           20133,
		No:            4260,
		Abstain:       30,
		VotingStarted: 1596240000,
		LockedIn:      544383,
	},
	"treasurwy": {
		Yes:     20133,
		No:      4260,
		Abstain: 30,
	},
}

func (pg *PgDb) AgendasVotesSummary(agendaID string) (summary *dbhelper.AgendaSummary, err error) {
	return nil, nil
}
