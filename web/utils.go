// TODO: move to shared package
package web

import (
	"io"
	"net/http"
)

// CommonData grabs the common page data that is available to every page.
// This is particularly useful for extras.tmpl, parts of which
// are used on every page
func (s *Server) CommonData(r *http.Request) *CommonPageData {
	darkMode, err := r.Cookie(DarkModeCoookie)
	if err != nil && err != http.ErrNoCookie {
		log.Errorf("Cookie pdanalyticsDarkBG retrieval error: %v", err)
	}
	// return &CommonPageData{
	// 	Version:       version.Version(),
	// 	ChainParams:   s.params,
	// 	BlockTimeUnix: int64(params.TargetTimePerBlock.Seconds()),
	// 	//DevAddress:    exp.pageData.HomeInfo.DevAddress,
	// 	//NetName:       exp.NetName,
	// 	MenuItems: s.MenuItems,
	// 	Links:     ExplorerLinks,
	// 	Cookies: Cookies{
	// 		DarkMode: darkMode != nil && darkMode.Value == "1",
	// 	},
	// 	RequestURI: r.URL.RequestURI(),
	// }
	data := s.common

	data.RequestURI = r.URL.RequestURI()

	data.Cookies = Cookies{
		DarkMode: darkMode != nil && darkMode.Value == "1",
	}

	return &data
}

// StatusPage provides a page for displaying status messages and exception
// handling without redirecting. Be sure to return after calling StatusPage if
// this completes the processing of the calling http handler.
func (s *Server) StatusPage(w http.ResponseWriter, r *http.Request, code, message, additionalInfo string, sType ExpStatus) {
	commonPageData := s.CommonData(r)
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
