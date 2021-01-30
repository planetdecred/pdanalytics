// TODO: move to shared package
package web

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/txscript/v2"
	"github.com/planetdecred/pdanalytics/version"
)

// DevSubsidyAddress returns the development subsidy address for the specified
// network.
func DevSubsidyAddress(params *chaincfg.Params) (string, error) {
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

// CalculateHashRate calculates the hashrate from the difficulty value and
// the targetTimePerBlock in seconds. The hashrate returned is in form PetaHash
// per second (PH/s).
func CalculateHashRate(difficulty, targetTimePerBlock float64) float64 {
	return ((difficulty * math.Pow(2, 32)) / targetTimePerBlock) / 1000000
}

// CommonData grabs the common page data that is available to every page.
// This is particularly useful for extras.tmpl, parts of which
// are used on every page
func (s *Server) CommonData(r *http.Request, params *chaincfg.Params) *CommonPageData {
	darkMode, err := r.Cookie(DarkModeCoookie)
	if err != nil && err != http.ErrNoCookie {
		log.Errorf("Cookie pdanalyticsDarkBG retrieval error: %v", err)
	}
	return &CommonPageData{
		Version:       version.Version(),
		ChainParams:   params,
		BlockTimeUnix: int64(params.TargetTimePerBlock.Seconds()),
		//DevAddress:    exp.pageData.HomeInfo.DevAddress,
		//NetName:       exp.NetName,
		Links: ExplorerLinks,
		Cookies: Cookies{
			DarkMode: darkMode != nil && darkMode.Value == "1",
		},
		RequestURI: r.URL.RequestURI(),
	}
}

// StatusPage provides a page for displaying status messages and exception
// handling without redirecting. Be sure to return after calling StatusPage if
// this completes the processing of the calling http handler.
func (s *Server) StatusPage(w http.ResponseWriter, r *http.Request, code, message, additionalInfo string, sType ExpStatus, params *chaincfg.Params) {
	commonPageData := s.CommonData(r, params)
	if commonPageData == nil {
		// exp.blockData.GetTip likely failed due to empty DB.
		http.Error(w, "The database is initializing. Try again later.",
			http.StatusServiceUnavailable)
		return
	}
	str, err := s.Templates.Exec("status", struct {
		*CommonPageData
		StatusType     ExpStatus
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
	case ExpStatusDBTimeout:
		w.WriteHeader(http.StatusServiceUnavailable)
	case ExpStatusNotFound:
		w.WriteHeader(http.StatusNotFound)
	case ExpStatusFutureBlock:
		w.WriteHeader(http.StatusOK)
	case ExpStatusError:
		w.WriteHeader(http.StatusInternalServerError)
	// When blockchain sync is running, status 202 is used to imply that the
	// other requests apart from serving the status sync page have been received
	// and accepted but cannot be processed now till the sync is complete.
	case ExpStatusSyncing:
		w.WriteHeader(http.StatusAccepted)
	case ExpStatusNotSupported:
		w.WriteHeader(http.StatusUnprocessableEntity)
	case ExpStatusBadRequest:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	io.WriteString(w, str)
}
