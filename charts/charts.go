package charts

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/wire"

	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

type Charts struct {
	client         *dcrd.Dcrd
	server         *web.Server
	ChainParams    *chaincfg.Params
	premine        int64
	targetPoolSize int64
	reorgLock      sync.Mutex
}

func New(client *dcrd.Dcrd, webServer *web.Server) (*Charts, error) {
	chrt := &Charts{
		server: webServer,
		client: client,
	}

	hash, err := client.Rpc.GetBestBlockHash()
	if err != nil {
		return nil, err
	}
	blockHeader, err := client.Rpc.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	if err = chrt.ConnectBlock(blockHeader); err != nil {
		return nil, err
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

	if err := chrt.server.Templates.AddTemplate("charts"); err != nil {
		log.Errorf("Unable to create new html template: %v", err)
		return nil, err
	}

	chrt.client.Notif.RegisterBlockHandlerGroup(chrt.ConnectBlock)

	chrt.server.AddRoute("/charts", web.GET, chrt.charts)
	chrt.server.AddRoute("/api/charts/{chartDataType}", web.GET, chrt.ChartTypeData, web.ChartDataTypeCtx)

	return chrt, nil
}

func (ch *Charts) ConnectBlock(w *wire.BlockHeader) error {
	ch.reorgLock.Lock()
	defer ch.reorgLock.Unlock()

	hashes, err := ch.client.Rpc.LiveTickets()
	if err != nil {
		return err
	}
	ch.targetPoolSize = int64(len(hashes))

	return nil
}

//charts is the page handler for the "/charts" path
func (ch *Charts) charts(w http.ResponseWriter, r *http.Request) {
	ch.reorgLock.Lock()

	str, err := ch.server.Templates.ExecTemplateToString("charts", struct {
		*web.CommonPageData
		TargetPoolSize  int64
		Premine         int64
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: ch.server.CommonData(r),
		Premine:        ch.premine,
		TargetPoolSize: ch.targetPoolSize,
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
		log.Errorf(err.Error())
	}
}

func (c *Charts) ChartTypeData(w http.ResponseWriter, r *http.Request) {
	chartType := web.GetChartDataTypeCtx(r)

	bin := r.URL.Query().Get("bin")

	// Support the deprecated URL parameter "zoom".
	if bin == "" {
		bin = r.URL.Query().Get("zoom")
	}
	axis := r.URL.Query().Get("axis")

	//specify timeouts of
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://dcrdata.decred.org/api/chart/%s?bin=%s&axis=%s", chartType, bin, axis), nil)

	if err != nil {
		http.NotFound(w, r)
		log.Warnf(`Error fetching chart %s at bin level '%s': %v`, chartType, bin, err)
		return
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		http.NotFound(w, r)
		log.Warnf(`Error fetching chart %s at bin level '%s': %v`, chartType, bin, err)
		return
	}
	defer resp.Body.Close()
	chartData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Print(err.Error())
		log.Warnf(`Error fetching chart %s at bin level '%s': %v`, chartType, bin, err)
	}

	web.RenderJSONBytes(w, chartData)
}
