package politeia

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	piproposals "github.com/dmigwi/go-piparser/proposals"
	pitypes "github.com/dmigwi/go-piparser/proposals/types"
	"github.com/planetdecred/pdanalytics/dcrd"
	dbtypes "github.com/planetdecred/pdanalytics/gov/politeia/types"
	"github.com/planetdecred/pdanalytics/web"
)

// isPiparserRunning is the flag set when a Piparser instance is running.
const isPiparserRunning = uint32(1)

// piParserCounter is a counter that helps guarantee that only one instance
// of proposalsUpdateHandler can ever be running at any one given moment.
var piParserCounter uint32

type dataSource interface {
	RetrieveLastCommitTime() (time.Time, error)
	InsertProposal(ctx context.Context, token string, author string, commitSHA string,
		entryDate time.Time, isChecked bool) (int, error)
	InsertProposalVote(ctx context.Context, proposalID int, voteTicket, voteBit string, isChecked bool) (int, error)
	RetrieveProposalVotesData(ctx context.Context, proposalToken string) (*dbtypes.ProposalChartsData, error)
}

// ProposalsFetcher defines the interface of the proposals plug-n-play data source.
type ProposalsFetcher interface {
	UpdateSignal() <-chan struct{}
	ProposalsHistory() ([]*pitypes.History, error)
	ProposalsHistorySince(since time.Time) ([]*pitypes.History, error)
}

// lastSync defines the latest sync time for the proposal votes sync.
type lastSync struct {
	mtx      sync.RWMutex
	syncTime time.Time
}

type proposals struct {
	ctx           context.Context
	client        *dcrd.Dcrd
	server        *web.Server
	db            *ProposalDB
	politeiaURL   string
	dataSource    dataSource
	piparser      ProposalsFetcher
	proposalsSync lastSync
}

func NewProposals(ctx context.Context, client *dcrd.Dcrd, dataSource dataSource, 
	politeiaURL, dbPath, piPropRepoOwner, piPropRepoName, dataDir string,
	webServer *web.Server) (*proposals, error) {

	db, err := newProposalsDB(politeiaURL, dbPath)
	if err != nil {
		return nil, err
	}

	parser, err := piproposals.NewParser(piPropRepoOwner, piPropRepoName, dataDir)
	if err != nil {
		return nil, err
	}

	prop := &proposals{
		client:      client,
		server:      webServer,
		db:          db,
		politeiaURL: politeiaURL,
		piparser:    parser,
	}

	// TODO: templates and routed registration
	prop.server.AddMenuItem(web.MenuItem{
		Href:      "/proposals",
		HyperText: "Proposals",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Proposals",
		},
	})

	prop.server.Templates.AddTemplate("proposals")
	prop.server.AddRoute("/proposals", web.GET, prop.ProposalsPage)

	return prop, nil
}

func (prop *proposals) Start(ctx context.Context) {

	// Initiate the piparser handler here.
	prop.startPiparserHandler(ctx)
	// Retrieve newly added proposals and add them to the proposals db(storm).
	// Proposal db update is made asynchronously to ensure that the system works
	// even when the Politeia API endpoint set is down.
	go func() {
		if err := prop.db.CheckProposalsUpdates(); err != nil {
			log.Errorf("updating proposals db failed: %v", err)
		}
	}()

	// An error in fetching the updates should not stop the system
	// functionality since it could be attributed to the external systems used.
	log.Info("Running updates retrieval for Politeia's Proposals. Please wait...")

	// Fetch updates for Politiea's Proposal history(votes) data via the parser.
	commitsCount, err := prop.PiProposalsHistory(ctx)
	if err != nil {
		log.Errorf("chainDB.PiProposalsHistory failed : %v", err)
	} else {
		log.Infof("%d politeia's proposal (auxiliary db) commits were processed",
			commitsCount)
	}
}

// startPiparserHandler controls how piparser update handler will be initiated.
// This handler should to be run once only when the first sync after startup completes.
func (pgb *proposals) startPiparserHandler(ctx context.Context) {
	if atomic.CompareAndSwapUint32(&piParserCounter, 0, isPiparserRunning) {
		// Start the proposal updates handler async method.
		pgb.proposalsUpdateHandler(ctx)

		log.Info("Piparser instance to handle updates is now active")
	} else {
		log.Error("piparser instance is already running, another one cannot be activated")
	}
}

// proposalsUpdateHandler runs in the background asynchronous to retrieve the
// politeia proposal updates that the piparser tool signaled.
func (pgb *proposals) proposalsUpdateHandler(ctx context.Context) {
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
					pgb.proposalsUpdateHandler(ctx)
				case <-pgb.ctx.Done():
				}
			}
		}()
		for range pgb.piparser.UpdateSignal() {
			count, err := pgb.PiProposalsHistory(ctx)
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
func (pgb *proposals) LastPiParserSync() time.Time {
	pgb.proposalsSync.mtx.RLock()
	defer pgb.proposalsSync.mtx.RUnlock()
	return pgb.proposalsSync.syncTime
}

// PiProposalsHistory queries the politeia's proposal updates via the parser tool
// and pushes them to the proposals and proposal_votes tables.
func (pgb *proposals) PiProposalsHistory(ctx context.Context) (int64, error) {
	if pgb.piparser == nil {
		return -1, fmt.Errorf("invalid piparser instance was found")
	}

	pgb.proposalsSync.mtx.Lock()

	// set the sync time
	pgb.proposalsSync.syncTime = time.Now().UTC()

	pgb.proposalsSync.mtx.Unlock()

	var isChecked bool
	var proposalsData []*pitypes.History

	lastUpdate, err := pgb.dataSource.RetrieveLastCommitTime()
	switch {
	case err == sql.ErrNoRows:
		// No records exists yet fetch all the history.
		proposalsData, err = pgb.piparser.ProposalsHistory()

	case err != nil:
		return -1, fmt.Errorf("RetrieveLastCommitTime failed :%v", err)

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

			id, err := pgb.dataSource.InsertProposal(ctx, val.Token, entry.Author,
				entry.CommitSHA, entry.Date, isChecked)
			if err != nil {
				return -1, fmt.Errorf("InsertProposal failed: %v", err)
			}

			for _, vote := range val.VotesInfo {
				_, err = pgb.dataSource.InsertProposalVote(ctx, id, vote.Ticket,
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
func (pgb *proposals) ProposalVotes(ctx context.Context, proposalToken string) (*dbtypes.ProposalChartsData, error) {
	return pgb.dataSource.RetrieveProposalVotesData(ctx, proposalToken)
}
