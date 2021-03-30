package politeia

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/decred/dcrd/wire"
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
	InsertProposal(tokenHash, author, commit string, timestamp time.Time, checked bool) (uint64, error)
	InsertProposalVote(proposalRowID uint64, ticket, choice string, checked bool) (uint64, error)
	RetrieveProposalVotesData(ctx context.Context, proposalToken string) (*dbtypes.ProposalChartsData, error)
	ProposalVotes(ctx context.Context, proposalToken string) (*dbtypes.ProposalChartsData, error)
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
	reorgLock     sync.Mutex
	height        uint32
}

// Activate activates the proposal module.
// This may take some time and should be ran in a goroutine
func Activate(ctx context.Context, client *dcrd.Dcrd, dataSource dataSource,
	politeiaURL, dbPath, piPropRepoOwner, piPropRepoName, dataDir string,
	webServer *web.Server, dataMode, httpMode bool) error {

	prop := &proposals{
		client:      client,
		server:      webServer,
		dataSource:  dataSource,
		politeiaURL: politeiaURL,
	}

	hash, err := client.Rpc.GetBestBlockHash()
	if err != nil {
		return err
	}
	blockHeader, err := client.Rpc.GetBlockHeader(hash)
	if err != nil {
		return err
	}

	if err = prop.connectBlock(blockHeader); err != nil {
		return err
	}
	if err := prop.server.Templates.AddTemplate("proposal"); err != nil {
		return err
	}

	client.Notif.RegisterBlockHandlerGroup(prop.connectBlock)

	if httpMode {
		prop.server.AddRoute("/proposals", web.GET, prop.ProposalsPage)
		prop.server.AddRoute("/proposal/{proposalrefid}", web.GET, prop.ProposalPage, proposalPathCtx)
		prop.server.AddRoute("/api/proposal/{token}", web.GET, prop.getProposalChartData, proposalTokenCtx)

		if err := prop.server.Templates.AddTemplate("proposals"); err != nil {
			return err
		}
		if err := prop.server.Templates.AddTemplate("proposal"); err != nil {
			return err
		}

		prop.server.AddMenuItem(web.MenuItem{
			Href:      "/proposals",
			HyperText: "Proposals",
			Attributes: map[string]string{
				"class": "menu-item",
				"title": "Proposals",
			},
		})
	}

	db, err := newProposalsDB(politeiaURL, dbPath)
	if err != nil {
		return err
	}
	prop.db = db

	if dataMode {
		log.Info("Creating proposal perser. This may take some time")
		parser, err := piproposals.NewParser(piPropRepoOwner, piPropRepoName, dataDir)
		if err != nil {
			return err
		}

		if parser == nil {
			return errors.New("Unable to get proposal parser")
		}
		prop.piparser = parser

		log.Info("Proposal perser created. Starting handler...")
		prop.start(ctx)
	}

	return nil
}

func (prop *proposals) start(ctx context.Context) {

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
		log.Errorf("proposals.PiProposalsHistory failed : %v", err)
	} else {
		log.Infof("%d politeia's proposal (auxiliary db) commits were processed",
			commitsCount)
	}
}

func (prop *proposals) connectBlock(w *wire.BlockHeader) error {
	prop.reorgLock.Lock()
	defer prop.reorgLock.Unlock()
	prop.height = w.Height

	return nil
}

// startPiparserHandler controls how piparser update handler will be initiated.
// This handler should to be run once only when the first sync after startup completes.
func (prop *proposals) startPiparserHandler(ctx context.Context) {
	if atomic.CompareAndSwapUint32(&piParserCounter, 0, isPiparserRunning) {
		// Start the proposal updates handler async method.
		prop.proposalsUpdateHandler(ctx)

		log.Info("Piparser instance to handle updates is now active")
	} else {
		log.Error("piparser instance is already running, another one cannot be activated")
	}
}

// proposalsUpdateHandler runs in the background asynchronous to retrieve the
// politeia proposal updates that the piparser tool signaled.
func (prop *proposals) proposalsUpdateHandler(ctx context.Context) {
	// Do not initiate the async update if invalid or disabled piparser instance was found.
	if prop.piparser == nil {
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
					prop.proposalsUpdateHandler(ctx)
				case <-prop.ctx.Done():
				}
			}
		}()
		for range prop.piparser.UpdateSignal() {
			count, err := prop.PiProposalsHistory(ctx)
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
func (prop *proposals) LastPiParserSync() time.Time {
	prop.proposalsSync.mtx.RLock()
	defer prop.proposalsSync.mtx.RUnlock()
	return prop.proposalsSync.syncTime
}

// PiProposalsHistory queries the politeia's proposal updates via the parser tool
// and pushes them to the proposals and proposal_votes tables.
func (prop *proposals) PiProposalsHistory(ctx context.Context) (int64, error) {
	if prop.piparser == nil {
		return -1, fmt.Errorf("invalid piparser instance was found")
	}

	prop.proposalsSync.mtx.Lock()

	// set the sync time
	prop.proposalsSync.syncTime = time.Now().UTC()

	prop.proposalsSync.mtx.Unlock()

	var isChecked bool
	var proposalsData []*pitypes.History

	lastUpdate, err := prop.dataSource.RetrieveLastCommitTime()
	switch {
	case err == sql.ErrNoRows:
		// No records exists yet fetch all the history.
		proposalsData, err = prop.piparser.ProposalsHistory()

	case err != nil:
		return -1, fmt.Errorf("RetrieveLastCommitTime failed :%v", err)

	default:
		// Fetch the updates since the last insert only.
		proposalsData, err = prop.piparser.ProposalsHistorySince(lastUpdate)
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

			id, err := prop.dataSource.InsertProposal(val.Token, entry.Author, entry.CommitSHA, entry.Date, isChecked)
			if err != nil {
				return -1, fmt.Errorf("InsertProposal failed: %v", err)
			}

			for _, vote := range val.VotesInfo {
				_, err = prop.dataSource.InsertProposalVote(id, vote.Ticket,
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
func (prop *proposals) ProposalVotes(ctx context.Context, proposalToken string) (*dbtypes.ProposalChartsData, error) {
	return prop.dataSource.RetrieveProposalVotesData(ctx, proposalToken)
}
