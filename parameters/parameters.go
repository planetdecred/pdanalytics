package parameters

import (
	"io"
	"net/http"

	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type Parameters struct {
	server *web.Server
	client *dcrd.Dcrd
}

func New(client *dcrd.Dcrd, server *web.Server) (*Parameters, error) {
	prm := &Parameters{
		server: server,
		client: client,
	}

	prm.server.AddMenuItem(web.MenuItem{
		Href:      "/parameters",
		HyperText: "Parameters",
		Info:      "Network parameters",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Chain Parameters",
		},
	})

	err := prm.server.Templates.AddTemplate("parameters")
	if err != nil {
		return nil, err
	}

	prm.server.AddRoute("/parameters", web.GET, prm.handle)

	return prm, nil
}

func (prm *Parameters) handle(w http.ResponseWriter, r *http.Request) {
	params := prm.client.Params
	addrPrefix := AddressPrefixes(params)
	actualTicketPoolSize := int64(params.TicketPoolSize * params.TicketsPerBlock)

	// exp.pageData.RLock()
	// var maxBlockSize int64
	// if exp.pageData.BlockchainInfo != nil {
	// 	maxBlockSize = exp.pageData.BlockchainInfo.MaxBlockSize
	// } else {
	// 	maxBlockSize = int64(params.MaximumBlockSizes[0])
	// }
	// exp.pageData.RUnlock()

	maxBlockSize := int64(params.MaximumBlockSizes[0])

	type ExtendedParams struct {
		MaximumBlockSize     int64
		ActualTicketPoolSize int64
		AddressPrefix        []AddrPrefix
	}

	str, err := prm.server.Templates.ExecTemplateToString("parameters", struct {
		*web.CommonPageData
		ExtendedParams
	}{
		CommonPageData: prm.server.CommonData(r),
		ExtendedParams: ExtendedParams{
			MaximumBlockSize:     maxBlockSize,
			AddressPrefix:        addrPrefix,
			ActualTicketPoolSize: actualTicketPoolSize,
		},
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		prm.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}
