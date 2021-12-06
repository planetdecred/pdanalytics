package treasury

import (
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/planetdecred/pdanalytics/web"
)

type Treasury struct {
	server *web.Server
	client *Client
	xcBot  *exchanges.ExchangeBot
	aPIURL string
}

func Activate(webServer *web.Server, xcBot *exchanges.ExchangeBot, apiurl string) error {
	client := NewClient()
	treasury := &Treasury{
		server: webServer,
		client: client,
		xcBot:  xcBot,
		aPIURL: apiurl,
	}

	treasury.server.AddMenuItem(web.MenuItem{
		Href:      "/treasury",
		HyperText: "Treasury",
		Info:      "Treasury transaction data.",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Treasury transaction data",
		},
	})

	treasury.server.AddRoute("/treasury", web.GET, treasury.TreasuryPage)
	treasury.server.AddRoute("/treasurytable", web.GET, treasury.TreasuryTable)

	err := treasury.server.Templates.AddTemplate("treasury")
	if err != nil {
		return err
	}
	err = treasury.server.Templates.AddTemplate("treasurytable")
	if err != nil {
		return err
	}

	return nil
}
