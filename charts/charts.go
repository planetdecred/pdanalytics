package charts

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrdata/exchanges/v2"

	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type Charts struct {
	client   *dcrd.Dcrd
	server   *web.Server
	xcBot    *exchanges.ExchangeBot
	pageData *web.PageData

	ChainParams    *chaincfg.Params
	targetPoolSize int64
	premine        int64

	reorgLock sync.Mutex
}

func New(client *dcrd.Dcrd, webServer *web.Server, xcBot *exchanges.ExchangeBot) (*Charts, error) {
	chrt := &Charts{
		server: webServer,
		xcBot:  xcBot,
		client: client,
	}

	chrt.server.AddMenuItem(web.MenuItem{
		Href:      "/charts",
		HyperText: "Charts",
		Info:      "Charts",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Charts",
		},
	})

	chrt.server.Templates.AddTemplate("charts")

	webServer.AddRoute("/charts", web.GET, chrt.charts)

	//TODO: Review this
	webServer.AddRoute("/api/chart/{chartDataType}", web.GET, chrt.ChartTypeData, web.ChartDataTypeCtx)

	return chrt, nil
}

//charts is the page handler for the "/charts" path
func (ch *Charts) charts(w http.ResponseWriter, r *http.Request) {
	ch.reorgLock.Lock()
	tpSize := ch.pageData.HomeInfo.PoolInfo.Target

	str, err := ch.server.Templates.ExecTemplateToString("charts", struct {
		*web.CommonPageData
		TargetPoolSize  uint32
		Premine         int64
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: ch.server.CommonData(r),
		Premine:        ch.premine,
		TargetPoolSize: tpSize,
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Charts",
				Active:    true,
			},
		},
	})
	ch.reorgLock.Unlock()

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		ch.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

func (c *Charts) ChartTypeData(w http.ResponseWriter, r *http.Request) {
	chartType := web.GetChartDataTypeCtx(r)
	bin := r.URL.Query().Get("bin")

	axis := r.URL.Query().Get("axis")

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://dcrdata.decred.org/api/chart/%s?bin=%s&axis=%s", chartType, bin, axis), nil)

	// chartData, err := c.charts.Chart(chartType, bin, axis)
	if err != nil {
		http.NotFound(w, r)
		log.Warnf(`Error fetching chart %s at bin level '%s': %v`, chartType, bin, err)
		return
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)

	defer resp.Body.Close()
	chartData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Print(err.Error())
	}

	web.RenderJSONBytes(w, chartData)
}

/* func (chrt *Charts) ConnectBlock(w *wire.BlockHeader) error {
	chrt.reorgLock.Lock()
	defer chrt.reorgLock.Unlock()

}
*/
