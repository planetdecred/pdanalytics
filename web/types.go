package web

import (
	"github.com/go-chi/chi"
)

type Server struct {
	webMux      *chi.Mux
	cfg         Config
	Templates   *Templates
	MenuItems   []MenuItem
	routes      map[string]route
	routeGroups []routeGroup
}

// Links to be passed with common page data.
type Links struct {
	CoinbaseComment string
	POSExplanation  string
	APIDocs         string
	InsightAPIDocs  string
	Github          string
	License         string
	NetParams       string
	DownloadLink    string
	// Testnet and below are set via dcrdata config.
	Testnet       string
	Mainnet       string
	TestnetSearch string
	MainnetSearch string
	OnionURL      string
}

type MenuItem struct {
	Href       string
	HyperText  string
	Attributes map[string]string
}

// Cookies contains information from the request cookies.
type Cookies struct {
	DarkMode bool
}
