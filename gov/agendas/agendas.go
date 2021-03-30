package agendas

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type agendas struct {
	ctx           context.Context
	client        *dcrd.Dcrd
	server        *web.Server
	reorgLock     sync.Mutex
	height        uint32
	agendasSource *AgendaDB
	voteTracker   *VoteTracker
}

// Activate activates the proposal module.
// This may take some time and should be ran in a goroutine
func Activate(ctx context.Context, client *dcrd.Dcrd, voteCounter voteCounter,
	agendasDBFileName, dataDir string, webServer *web.Server, dataMode, httpMode, simNet bool) error {

	agendaDB, err := NewAgendasDB(client.Rpc, filepath.Join(dataDir, agendasDBFileName))
	if err != nil {
		return fmt.Errorf("failed to create new agendas db instance: %v", err)
	}

	// A vote tracker tracks current block and stake versions and votes. Only
	// initialize the vote tracker if not on simnet. nil tracker is a sentinel
	// value throughout.
	var tracker *VoteTracker
	if !simNet {
		tracker, err = NewVoteTracker(client.Params, client.Rpc, voteCounter)
		if err != nil {
			return fmt.Errorf("Unable to initialize vote tracker: %v", err)
		}
	}

	agen := &agendas{
		client:        client,
		server:        webServer,
		agendasSource: agendaDB,
		voteTracker:   tracker,
	}

	if httpMode {
		agen.server.AddRoute("/agendas", web.GET, agen.AgendasPage)

		if err := agen.server.Templates.AddTemplate("agendas"); err != nil {
			return err
		}

		agen.server.AddMenuItem(web.MenuItem{
			Href:      "/agendas",
			HyperText: "Agendas",
			Attributes: map[string]string{
				"class": "menu-item",
				"title": "Agendas",
			},
		})
	}

	if dataMode {
		// The proposals and agenda db updates are run after the db indexing.
		// Retrieve blockchain deployment updates and add them to the agendas db.
		if err = agendaDB.UpdateAgendas(); err != nil {
			return fmt.Errorf("updating agendas db failed: %v", err)
		}
	}

	return nil
}
