package treasury

import (
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/web"
)

type Treasury struct {
	server *web.Server
	client *Client
	xcBot  *exchanges.ExchangeBot
}

func Activate(webServer *web.Server, xcBot *exchanges.ExchangeBot) error {
	client := NewClient()
	treasury := &Treasury{
		server: webServer,
		client: client,
		xcBot:  xcBot,
	}

	treasury.server.AddRoute("/treasury", web.GET, treasury.TreasuryPage)
	treasury.server.AddRoute("/treasurytable", web.GET, treasury.TreasuryTable)
	return nil
}
