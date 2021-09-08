package postgres

import (
	"context"
	"database/sql"
	"time"

	dbtypes "github.com/planetdecred/pdanalytics/gov/politeia/types"
)

const (
	createProposalTableScript = `CREATE TABLE IF NOT EXISTS proposals (
		id SERIAL PRIMARY KEY,
		token TEXT NOT NULL,
		author TEXT,
		commit_sha TEXT NOT NULL,
		time TIMESTAMPTZ
	);`

	IndexProposalsTableOnToken = `CREATE UNIQUE INDEX uix_proposals` +
		` ON proposals(token, time);`

	createProposalVotesTableScript = `CREATE TABLE IF NOT EXISTS proposal_votes (
		id SERIAL PRIMARY KEY,
		proposals_row_id INT8,
		ticket TEXT NOT NULL,
		choice TEXT NOT NULL
	);`

	IndexProposalVotesTableOnProposalsID = `CREATE INDEX uix_proposal_votes ` +
		` ON proposal_votes(proposals_row_id);`

	DeindexProposalVotesTableOnProposalsID = `DROP INDEX uix_proposal_votes CASCADE;`

	// Insert

	insertProposalsRow = `INSERT INTO proposals (token, author, commit_sha, time)
		VALUES ($1, $2, $3, $4) `

	InsertProposalsRow = insertProposalsRow + `RETURNING id;`

	UpsertProposalsRow = insertProposalsRow + `ON CONFLICT (token, time)
		DO UPDATE SET commit_sha = $3, time = $4  RETURNING id;`

	// Select

	SelectProposalsLastCommitTime = `Select time
		FROM proposals
		ORDER BY time DESC
		LIMIT 1;`

	// Proposal Votes table

	CreateProposalVotesTable = `CREATE TABLE IF NOT EXISTS proposal_votes (
		id SERIAL PRIMARY KEY,
		proposals_row_id INT8,
		ticket TEXT NOT NULL,
		choice TEXT NOT NULL
	);`

	// Insert

	insertProposalVotesRow = `INSERT INTO proposal_votes (proposals_row_id, ticket, choice)
		VALUES ($1, $2, $3) `

	InsertProposalVotesRow = insertProposalVotesRow + `RETURNING id;`

	// Select

	SelectProposalVotesChartData = `SELECT proposals.time,
		COUNT(CASE WHEN proposal_votes.choice = 'No' THEN 1 ElSE NULL END) as no,
		COUNT(CASE WHEN proposal_votes.choice = 'Yes' THEN 1 ElSE NULL END) as yes
		FROM proposal_votes
		INNER JOIN proposals on proposals.id = proposal_votes.proposals_row_id
		WHERE proposals.token = $1
		GROUP BY proposals.time
		ORDER BY proposals.time;`
)

func (pg *PgDb) RetrieveLastCommitTime() (entryTime time.Time, err error) {
	rows := pg.db.QueryRow(SelectProposalsLastCommitTime)
	err = rows.Scan(&entryTime)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// MakeProposalsInsertStatement returns the appropriate proposals insert statement for
// the desired conflict checking and handling behavior. See the description of
// MakeTicketInsertStatement for details.
func MakeProposalsInsertStatement(checked bool) string {
	if checked {
		return UpsertProposalsRow
	}
	return InsertProposalsRow
}

// InsertProposal adds the proposal details per commit to the proposal table.
func (pg *PgDb) InsertProposal(tokenHash, author, commit string, timestamp time.Time, checked bool) (uint64, error) {
	insertStatement := MakeProposalsInsertStatement(checked)
	var id uint64
	err := pg.db.QueryRow(insertStatement, tokenHash, author, commit, timestamp).Scan(&id)
	return id, err
}

// InsertProposalVote add the proposal votes entries to the proposal_votes table.
func (pg *PgDb) InsertProposalVote(proposalRowID uint64, ticket, choice string, checked bool) (uint64, error) {
	var id uint64
	err := pg.db.QueryRow(InsertProposalVotesRow, proposalRowID, ticket, choice).Scan(&id)
	return id, err
}

// retrieveProposalVotesData returns the vote data associated with the provided
// proposal token.
func (pg *PgDb) RetrieveProposalVotesData(ctx context.Context,
	proposalToken string) (*dbtypes.ProposalChartData, error) {
	rows, err := pg.db.QueryContext(ctx, SelectProposalVotesChartData, proposalToken)
	if err != nil {
		return nil, err
	}

	defer closeRows(rows)

	data := new(dbtypes.ProposalChartData)
	for rows.Next() {
		var yes, no uint64
		var timestamp time.Time

		if err = rows.Scan(&timestamp, &no, &yes); err != nil {
			return nil, err
		}

		data.No = append(data.No, no)
		data.Yes = append(data.Yes, yes)
		data.Time = append(data.Time, timestamp.Unix())
	}
	err = rows.Err()

	return data, err
}

// ProposalVotes retrieves all the votes data associated with the provided token.
func (pg *PgDb) ProposalVotes(ctx context.Context, proposalToken string) (*dbtypes.ProposalChartData, error) {
	ctx, cancel := context.WithTimeout(ctx, pg.queryTimeout)
	defer cancel()
	chartsData, err := pg.RetrieveProposalVotesData(ctx, proposalToken)
	return chartsData, pg.replaceCancelError(err)
}

// closeRows closes the input sql.Rows, logging any error.
func closeRows(rows *sql.Rows) {
	if e := rows.Close(); e != nil {
		log.Errorf("Close of Query failed: %v", e)
	}
}
