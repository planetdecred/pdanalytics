package politeia

import (
	"time"

	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type dataSource interface {
	LastPiParserSync() time.Time
}

type proposals struct {
	client      *dcrd.Dcrd
	server      *web.Server
	db          *ProposalDB
	politeiaURL string
	dataSource  dataSource
}

func NewProposals(client *dcrd.Dcrd, politeiaURL string, dbPath string, dataSource dataSource,
	webServer *web.Server) (*proposals, error) {
	db, err := newProposalsDB(politeiaURL, dbPath)
	if err != nil {
		return nil, err
	}
	prop := &proposals{
		client:      client,
		server:      webServer,
		db:          db,
		politeiaURL: politeiaURL,
	}

	// TODO: templates and routed registration

	return prop, nil
}
