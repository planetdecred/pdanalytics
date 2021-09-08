package politeia

import (
	"context"
	"sync"
	"time"

	"github.com/decred/dcrd/wire"
	"github.com/planetdecred/pdanalytics/dcrd"
	dbtypes "github.com/planetdecred/pdanalytics/gov/politeia/types"
	"github.com/planetdecred/pdanalytics/web"
)

type dataSource interface {
	RetrieveLastCommitTime() (time.Time, error)
	InsertProposal(tokenHash, author, commit string, timestamp time.Time, checked bool) (uint64, error)
	InsertProposalVote(proposalRowID uint64, ticket, choice string, checked bool) (uint64, error)
	RetrieveProposalVotesData(ctx context.Context, proposalToken string) (*dbtypes.ProposalChartData, error)
	ProposalVotes(ctx context.Context, proposalToken string) (*dbtypes.ProposalChartData, error)
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
	db            *ProposalsDB
	politeiaURL   string
	dataSource    dataSource
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
		prop.server.AddRoute("/proposal/{token}", web.GET, prop.ProposalPage, proposalTokenCtx)
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
			Info:      "Governance Proposals",
			Attributes: map[string]string{
				"class": "menu-item",
				"title": "Proposals",
			},
		})
	}

	db, err := NewProposalsDB(politeiaURL, dbPath)
	if err != nil {
		return err
	}
	prop.db = db

	prop.start(ctx)

	return nil
}

func (prop *proposals) start(ctx context.Context) {
	// Retrieve newly added proposals and add them to the proposals db(storm).
	// Proposal db update is made asynchronously to ensure that the system works
	// even when the Politeia API endpoint set is down.
	go func() {
		if err := prop.db.ProposalsSync(); err != nil {
			log.Errorf("updating proposals db failed: %v", err)
		}
	}()

	// Retrieve newly added proposals and add them to the proposals db(storm),
	// Every 5 minutes.
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := prop.db.ProposalsSync(); err != nil {
					log.Errorf("updating proposals db failed: %v", err)
				}
			case <-ctx.Done():
				log.Info("Shutting down Proposal Syncer")
				ticker.Stop()
				return
			}
		}
	}()
}

func (prop *proposals) connectBlock(w *wire.BlockHeader) error {
	prop.reorgLock.Lock()
	defer prop.reorgLock.Unlock()
	prop.height = w.Height

	return nil
}
