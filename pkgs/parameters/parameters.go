package parameters

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/rpcclient/v5"
	"github.com/decred/dcrd/txscript/v2"

	"github.com/planetdecred/pdanalytics/web"
)

type parameter struct {
	templates *web.Templates
	webServer *web.Server

	pageData *web.PageData

	ChainParams      *chaincfg.Params
	Version          string
	NetName          string
	MeanVotingBlocks int64

	dcrdChainSvr *rpcclient.Client
}

func New(dcrdClient *rpcclient.Client, webServer *web.Server, params *chaincfg.Params) (*parameter, error) {
	exp := &parameter{
		templates:    webServer.Templates,
		webServer:    webServer,
		ChainParams:  params,
		dcrdChainSvr: dcrdClient,
	}

	tmpls := []string{"parameters"}

	for _, name := range tmpls {
		if err := exp.templates.AddTemplate(name); err != nil {
			log.Errorf("Unable to create new html template: %v", err)
			return nil, err
		}
	}

	exp.webServer.AddMenuItem(web.MenuItem{
		Href:      "/parameters",
		HyperText: "Parameters",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Chain Parameters",
		},
	})

	exp.webServer.AddMenuItem(web.MenuItem{})

	// Development subsidy address of the current network
	devSubsidyAddress, err := devSubsidyAddress(params)
	if err != nil {
		log.Warnf("parameters.New: %v", err)
		return nil, err
	}
	log.Debugf("Organization address: %s", devSubsidyAddress)

	exp.pageData = &web.PageData{
		BlockInfo: new(web.BlockInfo),
		HomeInfo: &web.HomeInfo{
			DevAddress: devSubsidyAddress,
			Params: web.ChainParams{
				WindowSize:       exp.ChainParams.StakeDiffWindowSize,
				RewardWindowSize: exp.ChainParams.SubsidyReductionInterval,
				BlockTime:        exp.ChainParams.TargetTimePerBlock.Nanoseconds(),
				MeanVotingBlocks: exp.MeanVotingBlocks,
			},
			PoolInfo: web.TicketPoolInfo{
				Target: uint32(exp.ChainParams.TicketPoolSize * exp.ChainParams.TicketsPerBlock),
			},
		},
	}

	webServer.AddRoute("/parameters", web.GET, exp.handle)

	return exp, nil
}

// DevSubsidyAddress returns the development subsidy address for the specified
// network.
func devSubsidyAddress(params *chaincfg.Params) (string, error) {
	var devSubsidyAddress string
	var err error
	switch params.Name {
	case "testnet2":
		// TestNet2 uses an invalid organization PkScript
		devSubsidyAddress = "TccTkqj8wFqrUemmHMRSx8SYEueQYLmuuFk"
		err = fmt.Errorf("testnet2 has invalid project fund script")
	default:
		_, devSubsidyAddresses, _, err0 := txscript.ExtractPkScriptAddrs(
			params.OrganizationPkScriptVersion, params.OrganizationPkScript, params)
		if err0 != nil || len(devSubsidyAddresses) != 1 {
			err = fmt.Errorf("failed to decode dev subsidy address: %v", err0)
		} else {
			devSubsidyAddress = devSubsidyAddresses[0].String()
		}
	}
	return devSubsidyAddress, err
}

// AddrPrefix represent the address name it's prefix and description
type AddrPrefix struct {
	Name        string
	Prefix      string
	Description string
}

// AddressPrefixes generates an array AddrPrefix by using chaincfg.Params
// TODO: move to shared package
func AddressPrefixes(params *chaincfg.Params) []AddrPrefix {
	Descriptions := []string{"P2PK address",
		"P2PKH address prefix. Standard wallet address. 1 public key -> 1 private key",
		"Ed25519 P2PKH address prefix",
		"secp256k1 Schnorr P2PKH address prefix",
		"P2SH address prefix",
		"WIF private key prefix",
		"HD extended private key prefix",
		"HD extended public key prefix",
	}
	Name := []string{"PubKeyAddrID",
		"PubKeyHashAddrID",
		"PKHEdwardsAddrID",
		"PKHSchnorrAddrID",
		"ScriptHashAddrID",
		"PrivateKeyID",
		"HDPrivateKeyID",
		"HDPublicKeyID",
	}

	MainnetPrefixes := []string{"Dk", "Ds", "De", "DS", "Dc", "Pm", "dprv", "dpub"}
	TestnetPrefixes := []string{"Tk", "Ts", "Te", "TS", "Tc", "Pt", "tprv", "tpub"}
	SimnetPrefixes := []string{"Sk", "Ss", "Se", "SS", "Sc", "Ps", "sprv", "spub"}

	name := params.Name
	var netPrefixes []string
	if name == "mainnet" {
		netPrefixes = MainnetPrefixes
	} else if strings.HasPrefix(name, "testnet") {
		netPrefixes = TestnetPrefixes
	} else if name == "simnet" {
		netPrefixes = SimnetPrefixes
	} else {
		return nil
	}

	addrPrefix := make([]AddrPrefix, 0, len(Descriptions))
	for i, desc := range Descriptions {
		addrPrefix = append(addrPrefix, AddrPrefix{
			Name:        Name[i],
			Description: desc,
			Prefix:      netPrefixes[i],
		})
	}
	return addrPrefix
}

// commonData grabs the common page data that is available to every page.
// This is particularly useful for extras.tmpl, parts of which
// are used on every page
func (exp *parameter) commonData(r *http.Request) *web.CommonPageData {

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

func (exp parameter) handle(w http.ResponseWriter, r *http.Request) {
	params := exp.ChainParams
	addrPrefix := AddressPrefixes(params)
	actualTicketPoolSize := int64(params.TicketPoolSize * params.TicketsPerBlock)

	exp.pageData.RLock()
	var maxBlockSize int64
	if exp.pageData.BlockchainInfo != nil {
		maxBlockSize = exp.pageData.BlockchainInfo.MaxBlockSize
	} else {
		maxBlockSize = int64(params.MaximumBlockSizes[0])
	}
	exp.pageData.RUnlock()

	type ExtendedParams struct {
		MaximumBlockSize     int64
		ActualTicketPoolSize int64
		AddressPrefix        []AddrPrefix
	}

	str, err := exp.templates.ExecTemplateToString("parameters", struct {
		*web.CommonPageData
		ExtendedParams
	}{
		CommonPageData: exp.commonData(r),
		ExtendedParams: ExtendedParams{
			MaximumBlockSize:     maxBlockSize,
			AddressPrefix:        addrPrefix,
			ActualTicketPoolSize: actualTicketPoolSize,
		},
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// StatusPage provides a page for displaying status messages and exception
// handling without redirecting. Be sure to return after calling StatusPage if
// this completes the processing of the calling http handler.
func (exp *parameter) StatusPage(w http.ResponseWriter, r *http.Request, code, message, additionalInfo string, sType web.ExpStatus) {
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
