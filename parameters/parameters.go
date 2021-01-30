package parameters

import (
	"io"
	"net/http"

	"github.com/planetdecred/pdanalytics/base"
	"github.com/planetdecred/pdanalytics/web"
)

type Parameters struct {
	*base.Base
}

func New(b *base.Base) (*Parameters, error) {
	prm := &Parameters{
		Base: b,
	}

	prm.WebServer.AddMenuItem(web.MenuItem{
		Href:      "/parameters",
		HyperText: "Parameters",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Chain Parameters",
		},
	})

	prm.WebServer.AddMenuItem(web.MenuItem{})

	// Development subsidy address of the current network
	// devSubsidyAddress, err := web.DevSubsidyAddress(b.Params)
	// if err != nil {
	// 	log.Warnf("parameters.New: %v", err)
	// 	return nil, err
	// }
	// log.Debugf("Organization address: %s", devSubsidyAddress)

	// exp.pageData = &web.PageData{
	// 	BlockInfo: new(web.BlockInfo),
	// 	HomeInfo: &web.HomeInfo{
	// 		DevAddress: devSubsidyAddress,
	// 		Params: web.ChainParams{
	// 			WindowSize:       exp.ChainParams.StakeDiffWindowSize,
	// 			RewardWindowSize: exp.ChainParams.SubsidyReductionInterval,
	// 			BlockTime:        exp.ChainParams.TargetTimePerBlock.Nanoseconds(),
	// 			MeanVotingBlocks: exp.MeanVotingBlocks,
	// 		},
	// 		PoolInfo: web.TicketPoolInfo{
	// 			Target: uint32(exp.ChainParams.TicketPoolSize * exp.ChainParams.TicketsPerBlock),
	// 		},
	// 	},
	// }

	err := prm.WebServer.Templates.AddTemplate("parameters")
	if err != nil {
		return nil, err
	}

	prm.WebServer.AddRoute("/parameters", web.GET, prm.handle)

	return prm, nil
}

func (prm *Parameters) handle(w http.ResponseWriter, r *http.Request) {
	params := prm.Params
	addrPrefix := web.AddressPrefixes(params)
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
		AddressPrefix        []web.AddrPrefix
	}

	str, err := prm.WebServer.Templates.ExecTemplateToString("parameters", struct {
		*web.CommonPageData
		ExtendedParams
	}{
		CommonPageData: prm.WebServer.CommonData(r, prm.Params),
		ExtendedParams: ExtendedParams{
			MaximumBlockSize:     maxBlockSize,
			AddressPrefix:        addrPrefix,
			ActualTicketPoolSize: actualTicketPoolSize,
		},
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		prm.WebServer.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError, prm.Params)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}
