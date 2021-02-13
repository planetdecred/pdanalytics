package web

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/go-chi/chi"
	"github.com/planetdecred/pdanalytics/version"
)

const (
	DarkModeCoookie   = "pdanalyticsDarkBG"
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
	Github:          "https://github.com/planetdecred/pdanalytics",
	License:         "https://github.com/planetdecred/pdanalytics/blob/master/LICENSE",
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

// NewServer
func NewServer(cfg Config, mux *chi.Mux, params *chaincfg.Params) (*Server, error) {
	commonTemplates := []string{"layout"}
	templates := NewTemplates(cfg.Viewsfolder, cfg.ReloadHTML, commonTemplates, MakeTemplateFuncMap(params))
	templates.AddTemplate("status")
	
	s := &Server{
		webMux:      mux,
		cfg:         cfg,
		Templates:   templates,
		routes:      map[string]route{},
		routeGroups: []routeGroup{},
		common: CommonPageData{
			Version:       version.Version(),
			ChainParams:   params,
			BlockTimeUnix: int64(params.TargetTimePerBlock.Seconds()),
			//DevAddress:    exp.pageData.HomeInfo.DevAddress,
			//NetName:       exp.NetName,
			MenuItems: make([]MenuItem, 0),
			Links:     ExplorerLinks,
		},
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
	s.common.MenuItems = append(s.common.MenuItems, menuItem)
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
	r.With(CacheControl(cacheControlMaxAge)).Get(muxRoot, hf)
}
