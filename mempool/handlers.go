package mempool

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/planetdecred/pdanalytics/web"
)

const (
	mempoolDefaultChartDataType = "size"
)

func (c *Collector) mempoolPage(w http.ResponseWriter, r *http.Request) {
	mempoolData, err := c.fetchMempoolData(r)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		return
	}

	str, err := c.webServer.Templates.ExecTemplateToString("mempool", struct {
		*web.CommonPageData
		Mempool         map[string]interface{}
		BlockTime       float64
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: c.webServer.CommonData(r),
		Mempool:        mempoolData,
		BlockTime:      c.client.Params.MinDiffReductionTime.Seconds(),
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Mempool",
				Active:    true,
			},
		},
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		c.webServer.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

// getMempool is a handler for the "/getMempool" path.
func (s *Collector) getMempool(w http.ResponseWriter, r *http.Request) {
	data, err := s.fetchMempoolData(r)

	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		return
	}

	web.RenderJSON(w, data)
}

func (s *Collector) fetchMempoolData(req *http.Request) (map[string]interface{}, error) {
	req.ParseForm()
	page := req.FormValue("page")
	numberOfRows := req.FormValue("records-per-page")
	viewOption := req.FormValue("view-option")
	chartDataType := req.FormValue("chart-data-type")

	if chartDataType == "" {
		chartDataType = mempoolDefaultChartDataType
	}

	if viewOption == "" {
		viewOption = web.DefaultViewOption
	}

	var pageSize int
	numRows, err := strconv.Atoi(numberOfRows)
	switch {
	case err != nil || numRows <= 0:
		pageSize = web.DefaultPageSize
	case numRows > web.MaxPageSize:
		pageSize = web.MaxPageSize
	default:
		pageSize = numRows
	}

	pageToLoad, err := strconv.Atoi(page)
	if err != nil || pageToLoad <= 0 {
		pageToLoad = 1
	}

	offset := (pageToLoad - 1) * pageSize

	data := map[string]interface{}{
		"chartView":            true,
		"chartDataType":        chartDataType,
		"selectedViewOption":   viewOption,
		"pageSizeSelector":     web.PageSizeSelector,
		"selectedNumberOfRows": pageSize,
		"currentPage":          pageToLoad,
		"previousPage":         pageToLoad - 1,
		"totalPages":           0,
	}

	if viewOption == web.DefaultViewOption {
		return data, nil
	}

	ctx := req.Context()

	mempoolSlice, err := s.dataStore.Mempools(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	totalCount, err := s.dataStore.MempoolCount(ctx)
	if err != nil {
		return nil, err
	}

	if len(mempoolSlice) == 0 {
		data["message"] = fmt.Sprintf("Mempool %s", web.NoDataMessage)
		return data, nil
	}

	data["mempoolData"] = mempoolSlice
	data["totalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))

	totalTxLoaded := offset + len(mempoolSlice)
	if int64(totalTxLoaded) < totalCount {
		data["nextPage"] = pageToLoad + 1
	}

	return data, nil
}

// api/charts/mempool/{dataType}
func (c *Collector) chart(w http.ResponseWriter, r *http.Request) {
	dataType := web.GetChartDataTypeCtx(r)
	bin := r.URL.Query().Get("bin")

	chartData, err := c.dataStore.FetchEncodeChart(r.Context(), dataType, bin)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		log.Warnf(`Error fetching mempool %s chart: %v`, dataType, err)
		return
	}
	web.RenderJSONBytes(w, chartData)
}
