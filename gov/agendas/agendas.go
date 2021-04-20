package agendas

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/decred/dcrd/wire"
	"github.com/planetdecred/pdanalytics/dbhelper"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type dataSource interface {
	AgendasVotesSummary(agendaID string) (summary *dbhelper.AgendaSummary, err error)
}

type agendas struct {
	ctx           context.Context
	client        *dcrd.Dcrd
	server        *web.Server
	reorgLock     sync.Mutex
	height        uint32
	agendasSource *AgendaDB
	voteTracker   *VoteTracker
	dataSource    dataSource
}

// Activate activates the proposal module.
// This may take some time and should be ran in a goroutine
func Activate(ctx context.Context, client *dcrd.Dcrd, voteCounter voteCounter, dataSource dataSource,
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
		dataSource:    dataSource,
	}

	// move cache folder to the data directory
	// if the config file is missing, create the default
	pathNotExists := func(path string) bool {
		_, err := os.Stat(path)
		return os.IsNotExist(err)
	}

	if pathNotExists(path.Join(dataDir, "agendas_cache")) {
		log.Infof("creating %s for agendas cache", path.Join(dataDir, "agendas_cache"))
		if err = os.MkdirAll(path.Join(dataDir, "agendas_cache"), os.ModePerm); err != nil {
			return fmt.Errorf("Missing agendas cache dir and cannot create it - %s", err.Error())
		}
	}

	hash, err := client.Rpc.GetBestBlockHash()
	if err != nil {
		return err
	}
	blockHeader, err := client.Rpc.GetBlockHeader(hash)
	if err != nil {
		return err
	}

	if err = agen.ConnectBlock(blockHeader); err != nil {
		return err
	}

	if httpMode {
		agen.server.AddRoute("/agendas", web.GET, agen.AgendasPage)
		if err := agen.server.Templates.AddTemplate("agendas"); err != nil {
			return err
		}

		agen.server.AddRoute("/agenda/{id}", web.GET, agen.AgendaPage, agendaPathCtx)
		if err := agen.server.Templates.AddTemplate("agenda"); err != nil {
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

func copyFile(sourec, destination string) error {
	from, err := os.Open(sourec)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}

	return nil
}

func (ac *agendas) ConnectBlock(w *wire.BlockHeader) error {
	ac.reorgLock.Lock()
	defer ac.reorgLock.Unlock()
	ac.height = w.Height
	return nil
}
