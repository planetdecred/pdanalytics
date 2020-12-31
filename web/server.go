package web

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/decred/dcrd/chaincfg/v2"
	chainjson "github.com/decred/dcrd/rpc/jsonrpc/types/v2"
	"github.com/decred/dcrdata/explorer/types/v2"
	m "github.com/decred/dcrdata/middleware/v3"
	"github.com/go-chi/chi"
)

const (
	DarkModeCoookie   = "dcrdataDarkBG"
	DarkModeFormKey   = "darkmode"
	RequestURIFormKey = "requestURI"

	// Status page strings
	DefaultErrorCode    = "Something went wrong..."
	DefaultErrorMessage = "Try refreshing... it usually fixes things."
	PageDisabledCode    = "%s has been disabled for now."
	WrongNetwork        = "Wrong Network"
)

var ExplorerLinks = &Links{
	CoinbaseComment: "https://github.com/decred/dcrd/blob/2a18beb4d56fe59d614a7309308d84891a0cba96/chaincfg/genesis.go#L17-L53",
	POSExplanation:  "https://docs.decred.org/proof-of-stake/overview/",
	APIDocs:         "https://github.com/decred/dcrdata#apis",
	InsightAPIDocs:  "https://github.com/decred/dcrdata/blob/master/api/Insight_API_documentation.md",
	Github:          "https://github.com/decred/dcrdata",
	License:         "https://github.com/decred/dcrdata/blob/master/LICENSE",
	NetParams:       "https://github.com/decred/dcrd/blob/master/chaincfg/params.go",
	DownloadLink:    "https://decred.org/downloads/",
}

// ExpStatus defines the various status types supported by the system.
type ExpStatus string

// These are the explorer status messages used by the status page.
const (
	ExpStatusError          ExpStatus = "Error"
	ExpStatusNotFound       ExpStatus = "Not Found"
	ExpStatusFutureBlock    ExpStatus = "Future Block"
	ExpStatusNotSupported   ExpStatus = "Not Supported"
	ExpStatusBadRequest     ExpStatus = "Bad Request"
	ExpStatusNotImplemented ExpStatus = "Not Implemented"
	ExpStatusPageDisabled   ExpStatus = "Page Disabled"
	ExpStatusWrongNetwork   ExpStatus = "Wrong Network"
	ExpStatusDeprecated     ExpStatus = "Deprecated"
	ExpStatusSyncing        ExpStatus = "Blocks Syncing"
	ExpStatusDBTimeout      ExpStatus = "Database Timeout"
	ExpStatusP2PKAddress    ExpStatus = "P2PK Address Type"
)

func (e ExpStatus) IsNotFound() bool {
	return e == ExpStatusNotFound
}

func (e ExpStatus) IsWrongNet() bool {
	return e == ExpStatusWrongNetwork
}

func (e ExpStatus) IsP2PKAddress() bool {
	return e == ExpStatusP2PKAddress
}

func (e ExpStatus) IsFutureBlock() bool {
	return e == ExpStatusFutureBlock
}

func (e ExpStatus) IsSyncing() bool {
	return e == ExpStatusSyncing
}

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

// CommonPageData is the basis for data structs used for HTML templates.
// explorerUI.commonData returns an initialized instance or CommonPageData,
// which itself should be used to initialize page data template structs.
type CommonPageData struct {
	Tip           *types.WebBasicBlock
	Version       string
	ChainParams   *chaincfg.Params
	BlockTimeUnix int64
	DevAddress    string
	Links         *Links
	MenuItems     []MenuItem
	NetName       string
	Cookies       Cookies
	RequestURI    string
}

type PageData struct {
	sync.RWMutex
	BlockInfo      *types.BlockInfo
	BlockchainInfo *chainjson.GetBlockChainInfoResult
	HomeInfo       *types.HomeInfo
}

// A page number has the information necessary to create numbered pagination
// links.
type PageNumber struct {
	Active bool   `json:"active"`
	Link   string `json:"link"`
	Str    string `json:"str"`
}

func MakePageNumber(active bool, link, str string) PageNumber {
	return PageNumber{
		Active: active,
		Link:   link,
		Str:    str,
	}
}

type PageNumbers []PageNumber

// NewServer
func NewServer(cfg Config, mux *chi.Mux, chainParams *chaincfg.Params) (*Server, error) {
	commonTemplates := []string{"extras"}
	templates := newTemplates(cfg.Viewsfolder, cfg.ReloadHTML, commonTemplates, makeTemplateFuncMap(chainParams))
	s := &Server{
		webMux:      mux,
		cfg:         cfg,
		Templates:   &templates,
		MenuItems:   []MenuItem{},
		routes:      map[string]route{},
		routeGroups: []routeGroup{},
	}

	return s, nil
}

// Make the static assets available under a path with the given prefix.
func (s *Server) MountAssetPaths(pathPrefix string, publicFolder string) {
	if !strings.HasSuffix(pathPrefix, "/") {
		pathPrefix += "/"
	}
	if !strings.HasSuffix(publicFolder, "/") {
		publicFolder += "/"
	}

	s.webMux.Get(pathPrefix+"favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, publicFolder+"images/favicon/favicon.ico")
	})

	FileServer(s.webMux, pathPrefix+"js", publicFolder+"js", s.cfg.CacheControlMaxAge)
	FileServer(s.webMux, pathPrefix+"css", publicFolder+"css", s.cfg.CacheControlMaxAge)
	FileServer(s.webMux, pathPrefix+"fonts", publicFolder+"fonts", s.cfg.CacheControlMaxAge)
	FileServer(s.webMux, pathPrefix+"images", publicFolder+"images", s.cfg.CacheControlMaxAge)
	FileServer(s.webMux, pathPrefix+"dist", publicFolder+"dist", s.cfg.CacheControlMaxAge)
}

func (s *Server) AddMenuItem(menuItem MenuItem) {
	s.MenuItems = append(s.MenuItems, menuItem)
}

// FileServer conveniently sets up a http.FileServer handler to serve static
// files from path on the file system. Directory listings are denied, as are URL
// paths containing "..".
func FileServer(r chi.Router, pathRoot, fsRoot string, cacheControlMaxAge int64) {
	if strings.ContainsAny(pathRoot, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	// Define a http.HandlerFunc to serve files but not directory indexes.
	hf := func(w http.ResponseWriter, r *http.Request) {
		// Ensure the path begins with "/".
		upath := r.URL.Path
		if strings.Contains(upath, "..") {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		// Strip the path prefix and clean the path.
		upath = path.Clean(strings.TrimPrefix(upath, pathRoot))

		// Deny directory listings (http.ServeFile recognizes index.html and
		// attempts to serve the directory contents instead).
		if strings.HasSuffix(upath, "/index.html") {
			http.NotFound(w, r)
			return
		}

		// Generate the full file system path and test for existence.
		fullFilePath := filepath.Join(fsRoot, upath)
		fi, err := os.Stat(fullFilePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		// Deny directory listings
		if fi.IsDir() {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		http.ServeFile(w, r, fullFilePath)
	}

	// For the chi.Mux, make sure a path that ends in "/" and append a "*".
	muxRoot := pathRoot
	if pathRoot != "/" && pathRoot[len(pathRoot)-1] != '/' {
		r.Get(pathRoot, http.RedirectHandler(pathRoot+"/", 301).ServeHTTP)
		muxRoot += "/"
	}
	muxRoot += "*"

	// Mount the http.HandlerFunc on the pathRoot.
	r.With(m.CacheControl(cacheControlMaxAge)).Get(muxRoot, hf)
}
