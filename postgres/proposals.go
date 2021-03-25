package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/decred/dcrdata/db/dbtypes/v2"
)

// proposalsUpdateHandler runs in the background asynchronous to retrieve the
// politeia proposal updates that the piparser tool signaled.
func (pgb *ChainDB) proposalsUpdateHandler() {
	// Do not initiate the async update if invalid or disabled piparser instance was found.
	if pgb.piparser == nil {
		log.Error("invalid or disabled piparser instance found: proposals async update stopped")
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("recovered from piparser panic in proposalsUpdateHandler: %v", r)
				select {
				case <-time.NewTimer(time.Minute).C:
					log.Infof("attempting to restart proposalsUpdateHandler")
					pgb.proposalsUpdateHandler()
				case <-pgb.ctx.Done():
				}
			}
		}()
		for range pgb.piparser.UpdateSignal() {
			count, err := pgb.PiProposalsHistory()
			if err != nil {
				log.Error("pgb.PiProposalsHistory failed : %v", err)
			} else {
				log.Infof("%d politeia's proposal commits were processed", count)
			}
		}
	}()
}

// LastPiParserSync returns last time value when the piparser run sync on proposals
// and proposal_votes table.
func (pgb *ChainDB) LastPiParserSync() time.Time {
	pgb.proposalsSync.mtx.RLock()
	defer pgb.proposalsSync.mtx.RUnlock()
	return pgb.proposalsSync.syncTime
}

// PiProposalsHistory queries the politeia's proposal updates via the parser tool
// and pushes them to the proposals and proposal_votes tables.
func (pgb *ChainDB) PiProposalsHistory() (int64, error) {
	if pgb.piparser == nil {
		return -1, fmt.Errorf("invalid piparser instance was found")
	}

	pgb.proposalsSync.mtx.Lock()

	// set the sync time
	pgb.proposalsSync.syncTime = time.Now().UTC()

	pgb.proposalsSync.mtx.Unlock()

	var isChecked bool
	var proposalsData []*pitypes.History

	lastUpdate, err := retrieveLastCommitTime(pgb.db)
	switch {
	case err == sql.ErrNoRows:
		// No records exists yet fetch all the history.
		proposalsData, err = pgb.piparser.ProposalsHistory()

	case err != nil:
		return -1, fmt.Errorf("retrieveLastCommitTime failed :%v", err)

	default:
		// Fetch the updates since the last insert only.
		proposalsData, err = pgb.piparser.ProposalsHistorySince(lastUpdate)
		isChecked = true
	}

	if err != nil {
		return -1, fmt.Errorf("politeia proposals fetch failed: %v", err)
	}

	var commitsCount int64

	for _, entry := range proposalsData {
		if entry.CommitSHA == "" {
			// If missing commit sha ignore the entry.
			continue
		}

		// Multiple tokens votes data can be packed in a single Politeia's commit.
		for _, val := range entry.Patch {
			if val.Token == "" {
				// If missing token ignore it.
				continue
			}

			id, err := InsertProposal(pgb.db, val.Token, entry.Author,
				entry.CommitSHA, entry.Date, isChecked)
			if err != nil {
				return -1, fmt.Errorf("InsertProposal failed: %v", err)
			}

			for _, vote := range val.VotesInfo {
				_, err = InsertProposalVote(pgb.db, id, vote.Ticket,
					string(vote.VoteBit), isChecked)
				if err != nil {
					return -1, fmt.Errorf("InsertProposalVote failed: %v", err)
				}
			}
		}
		commitsCount++
	}

	return commitsCount, err
}

// ProposalVotes retrieves all the votes data associated with the provided token.
func (pgb *ChainDB) ProposalVotes(proposalToken string) (*dbtypes.ProposalChartsData, error) {
	ctx, cancel := context.WithTimeout(pgb.ctx, pgb.queryTimeout)
	defer cancel()
	chartsData, err := retrieveProposalVotesData(ctx, pgb.db, proposalToken)
	return chartsData, pgb.replaceCancelError(err)
}


// InsertProposal adds the proposal details per commit to the proposal table.
func InsertProposal(db *sql.DB, tokenHash, author, commit string,
	timestamp time.Time, checked bool) (uint64, error) {
	insertStatement := internal.MakeProposalsInsertStatement(checked)
	var id uint64
	err := db.QueryRow(insertStatement, tokenHash, author, commit, timestamp).Scan(&id)
	return id, err
}

// InsertProposalVote add the proposal votes entries to the proposal_votes table.
func InsertProposalVote(db *sql.DB, proposalRowID uint64, ticket, choice string,
	checked bool) (uint64, error) {
	var id uint64
	err := db.QueryRow(internal.InsertProposalVotesRow, proposalRowID, ticket, choice).Scan(&id)
	return id, err
}

// retrieveLastCommitTime returns the last commit timestamp whole proposal votes
// data was fetched and updated in both proposals and proposal_votes table.
func retrieveLastCommitTime(db *sql.DB) (timestamp time.Time, err error) {
	err = db.QueryRow(internal.SelectProposalsLastCommitTime).Scan(&timestamp)
	return
}

// retrieveProposalVotesData returns the vote data associated with the provided
// proposal token.
func retrieveProposalVotesData(ctx context.Context, db *sql.DB,
	proposalToken string) (*dbtypes.ProposalChartsData, error) {
	rows, err := db.QueryContext(ctx, internal.SelectProposalVotesChartData, proposalToken)
	if err != nil {
		return nil, err
	}

	defer closeRows(rows)

	data := new(dbtypes.ProposalChartsData)
	for rows.Next() {
		var yes, no uint64
		var timestamp time.Time

		if err = rows.Scan(&timestamp, &no, &yes); err != nil {
			return nil, err
		}

		data.No = append(data.No, no)
		data.Yes = append(data.Yes, yes)
		data.Time = append(data.Time, dbtypes.NewTimeDef(timestamp))
	}
	err = rows.Err()

	return data, err
}
