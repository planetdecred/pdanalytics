// TODO: move to shared package
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	retryDelay                  = 60 * time.Second
	maxRetryAttempts            = 3
	ChartViewOption             = "chart"
	DefaultViewOption           = ChartViewOption
	MaxPageSize                 = 250
	DefaultPageSize             = 20
	NoDataMessage               = "does not have data for the selected query option(s)."
)

var (
	PageSizeSelector = map[int]int{
		20:  20,
		30:  30,
		50:  50,
		100: 100,
		150: 150,
	}
)

// GetResponse attempts to collect json data from the given url string and decodes it into
// the destination
func GetResponse(ctx context.Context, client *http.Client, url string, destination interface{}) error {
	// if client has no timeout, set one
	if client.Timeout == time.Duration(0) {
		client.Timeout = 10 * time.Second
	}
	resp := new(http.Response)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	for i := 1; i <= maxRetryAttempts; i++ {
		res, err := client.Do(req)
		if err != nil {
			if res != nil {
				res.Body.Close()
			}
			if i == maxRetryAttempts {
				return err
			}
			time.Sleep(retryDelay)
			continue
		}
		resp = res
		break
	}

	err = json.NewDecoder(resp.Body).Decode(destination)
	if err != nil {
		return err
	}
	return nil
}

func AddParams(base string, params map[string]interface{}) (string, error) {
	var strBuilder strings.Builder

	_, err := strBuilder.WriteString(base)

	if err != nil {
		return base, err
	}

	strBuilder.WriteString("?")

	for param, value := range params {
		strBuilder.WriteString(param)
		strBuilder.WriteString("=")

		vType := reflect.TypeOf(value)
		switch vType.Kind() {
		case reflect.String:
			strBuilder.WriteString(reflect.ValueOf(value).String())
		case reflect.Int64, reflect.Int:
			strBuilder.WriteString(strconv.FormatInt(reflect.ValueOf(value).Int(), 10))
		case reflect.Float64:
			strBuilder.WriteString(strconv.FormatFloat(reflect.ValueOf(value).Float(), 'f', -1, 64))
		default:
			return strBuilder.String(), fmt.Errorf("unsupported type: %v", vType.Kind())
		}

		strBuilder.WriteString("&")
	}

	str := strBuilder.String()
	return str[:len(str)-1], nil
}

func NowUTC() time.Time {
	return time.Now().UTC()
}

func UnixTime(t int64) time.Time {
	return time.Unix(t, 0).UTC()
}

func DurationToString(duration time.Duration) string {
	duration = duration.Round(10 * time.Millisecond)
	return duration.String()
}

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
	data.Tip = &WebBasicBlock{}

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
