package base

import (
	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/rpcclient/v5"
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/web"
)

type Base struct {
	DcrdClient *rpcclient.Client
	WebServer  *web.Server
	XcBot      *exchanges.ExchangeBot
	Params     *chaincfg.Params
}
