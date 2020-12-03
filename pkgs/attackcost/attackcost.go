package attackcost

import (
	"io"
	"net/http"
	"sync"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/rpcclient/v5"
	"github.com/decred/dcrd/wire"
	"github.com/decred/dcrdata/db/dbtypes/v2"
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/decred/dcrdata/explorer/types/v2"
	"github.com/platnetdecred/pdanalytics/web"
)

type attackcost struct {
	templates *web.Templates
	webServer *web.Server
	xcBot     *exchanges.ExchangeBot

	pageData *web.PageData

	height          int64
	hashrate        float64
	ticketPrice     float64
	ticketPoolSize  int64
	ticketPoolValue float64
	coinSupply      int64

	ChainParams      *chaincfg.Params
	Version          string
	NetName          string
	MeanVotingBlocks int64

	dcrdChainSvr *rpcclient.Client
	reorgLock    sync.Mutex
}

func New(dcrdClient *rpcclient.Client, webServer *web.Server,
	xcBot *exchanges.ExchangeBot, params *chaincfg.Params) (*attackcost, error) {
	exp := &attackcost{
		templates:    webServer.Templates,
		webServer:    webServer,
		xcBot:        xcBot,
		ChainParams:  params,
		dcrdChainSvr: dcrdClient,
	}

	hash, err := dcrdClient.GetBestBlockHash()
	if err != nil {
		return nil, err
	}
	blockHeader, err := dcrdClient.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	if err = exp.ConnectBlock(blockHeader); err != nil {
		return nil, err
	}

	tmpls := []string{"attackcost"}

	for _, name := range tmpls {
		if err := exp.templates.AddTemplate(name); err != nil {
			log.Errorf("Unable to create new html template: %v", err)
			return nil, err
		}
	}

	exp.webServer.AddMenuItem(web.MenuItem{
		Href:      "/attack-cost",
		HyperText: "Attack Cost",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Decred Attack Cost",
		},
	})

	exp.webServer.AddMenuItem(web.MenuItem{})

	// Development subsidy address of the current network
	devSubsidyAddress, err := dbtypes.DevSubsidyAddress(params)
	if err != nil {
		log.Warnf("attackcost.New: %v", err)
		return nil, err
	}
	log.Debugf("Organization address: %s", devSubsidyAddress)

	exp.pageData = &web.PageData{
		BlockInfo: new(types.BlockInfo),
		HomeInfo: &types.HomeInfo{
			DevAddress: devSubsidyAddress,
			Params: types.ChainParams{
				WindowSize:       exp.ChainParams.StakeDiffWindowSize,
				RewardWindowSize: exp.ChainParams.SubsidyReductionInterval,
				BlockTime:        exp.ChainParams.TargetTimePerBlock.Nanoseconds(),
				MeanVotingBlocks: exp.MeanVotingBlocks,
			},
			PoolInfo: types.TicketPoolInfo{
				Target: exp.ChainParams.TicketPoolSize * exp.ChainParams.TicketsPerBlock,
			},
		},
	}

	webServer.AddRoute("/attack-cost", web.GET, exp.AttackCost)

	return exp, nil
}

func (exp *attackcost) ConnectBlock(w *wire.BlockHeader) error {
	exp.reorgLock.Lock()
	defer exp.reorgLock.Unlock()
	exp.height = int64(w.Height)
	hash := w.BlockHash()

	// Hashrate
	header, err := exp.dcrdChainSvr.GetBlockHeaderVerbose(&hash)
	if err != nil {
		return err
	}
	targetTimePerBlock := float64(exp.ChainParams.TargetTimePerBlock)
	exp.hashrate = dbtypes.CalculateHashRate(header.Difficulty, targetTimePerBlock)

	// Coin supply
	coinSupply, err := exp.dcrdChainSvr.GetCoinSupply()
	if err != nil {
		return err
	}
	exp.coinSupply = int64(coinSupply)

	// Stake difficulty (ticket price)
	stakeDiff, err := exp.dcrdChainSvr.GetStakeDifficulty()
	if err != nil {
		return err
	}
	exp.ticketPrice = stakeDiff.CurrentStakeDifficulty

	// Ticket pool info
	poolValue, err := exp.dcrdChainSvr.GetTicketPoolValue()
	if err != nil {
		return err
	}
	exp.ticketPoolValue = poolValue.ToCoin()
	hashes, err := exp.dcrdChainSvr.LiveTickets()
	if err != nil {
		return err
	}
	exp.ticketPoolSize = int64(len(hashes))

	return nil
}

// commonData grabs the common page data that is available to every page.
// This is particularly useful for extras.tmpl, parts of which
// are used on every page
func (exp *attackcost) commonData(r *http.Request) *web.CommonPageData {

	darkMode, err := r.Cookie(web.DarkModeCoookie)
	if err != nil && err != http.ErrNoCookie {
		log.Errorf("Cookie dcrdataDarkBG retrieval error: %v", err)
	}
	return &web.CommonPageData{
		Version:       exp.Version,
		ChainParams:   exp.ChainParams,
		BlockTimeUnix: int64(exp.ChainParams.TargetTimePerBlock.Seconds()),
		DevAddress:    exp.pageData.HomeInfo.DevAddress,
		NetName:       exp.NetName,
		Links:         web.ExplorerLinks,
		Cookies: web.Cookies{
			DarkMode: darkMode != nil && darkMode.Value == "1",
		},
		RequestURI: r.URL.RequestURI(),
		MenuItems:  exp.webServer.MenuItems,
	}
}

// AttackCost is the page handler for the "/attack-cost" path.
func (ac *attackcost) AttackCost(w http.ResponseWriter, r *http.Request) {
	price := 24.42
	if ac.xcBot != nil {
		if rate := ac.xcBot.Conversion(1.0); rate != nil {
			price = rate.Value
		}
	}

	ac.reorgLock.Lock()

	str, err := ac.templates.ExecTemplateToString("attackcost", struct {
		*web.CommonPageData
		HashRate        float64
		Height          int64
		DCRPrice        float64
		TicketPrice     float64
		TicketPoolSize  int64
		TicketPoolValue float64
		CoinSupply      int64
	}{
		CommonPageData:  ac.commonData(r),
		HashRate:        ac.hashrate,
		Height:          ac.height,
		DCRPrice:        price,
		TicketPrice:     ac.ticketPrice,
		TicketPoolSize:  ac.ticketPoolSize,
		TicketPoolValue: ac.ticketPoolValue,
		CoinSupply:      ac.coinSupply,
	})
	ac.reorgLock.Unlock()

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		ac.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

// StatusPage provides a page for displaying status messages and exception
// handling without redirecting. Be sure to return after calling StatusPage if
// this completes the processing of the calling http handler.
func (exp *attackcost) StatusPage(w http.ResponseWriter, r *http.Request, code, message, additionalInfo string, sType web.ExpStatus) {
	commonPageData := exp.commonData(r)
	if commonPageData == nil {
		// exp.blockData.GetTip likely failed due to empty DB.
		http.Error(w, "The database is initializing. Try again later.",
			http.StatusServiceUnavailable)
		return
	}
	str, err := exp.templates.Exec("status", struct {
		*web.CommonPageData
		StatusType     web.ExpStatus
		Code           string
		Message        string
		AdditionalInfo string
	}{
		CommonPageData: commonPageData,
		StatusType:     sType,
		Code:           code,
		Message:        message,
		AdditionalInfo: additionalInfo,
	})
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		str = "Something went very wrong if you can see this, try refreshing"
	}

	w.Header().Set("Content-Type", "text/html")
	switch sType {
	case web.ExpStatusDBTimeout:
		w.WriteHeader(http.StatusServiceUnavailable)
	case web.ExpStatusNotFound:
		w.WriteHeader(http.StatusNotFound)
	case web.ExpStatusFutureBlock:
		w.WriteHeader(http.StatusOK)
	case web.ExpStatusError:
		w.WriteHeader(http.StatusInternalServerError)
	// When blockchain sync is running, status 202 is used to imply that the
	// other requests apart from serving the status sync page have been received
	// and accepted but cannot be processed now till the sync is complete.
	case web.ExpStatusSyncing:
		w.WriteHeader(http.StatusAccepted)
	case web.ExpStatusNotSupported:
		w.WriteHeader(http.StatusUnprocessableEntity)
	case web.ExpStatusBadRequest:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	io.WriteString(w, str)
}
