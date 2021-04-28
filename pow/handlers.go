package pow

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/planetdecred/pdanalytics/web"
)

// /pow
func (c *Collector) powPage(w http.ResponseWriter, r *http.Request) {
	pows, err := c.fetchPoWData(r)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		return
	}

	str, err := c.server.Templates.ExecTemplateToString("pow", struct {
		*web.CommonPageData
		BreadcrumbItems []web.BreadcrumbItem
		Pow            map[string]interface{}
	}{
		CommonPageData: c.server.CommonData(r),
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Proof of Work mining pool data",
				Active:    true,
			},
		},
		Pow: pows,
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		c.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, err.Error(), web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

func (c *Collector) getFilteredPowData(w http.ResponseWriter, r *http.Request) {
	data, err := c.fetchPoWData(r)

	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		return
	}

	web.RenderJSON(w, data)
}

func (c *Collector) fetchPoWData(req *http.Request) (map[string]interface{}, error) {
	req.ParseForm()
	page := req.FormValue("page")
	selectedPow := req.FormValue("filter")
	selectedDataType := req.FormValue("data-type")
	numberOfRows := req.FormValue("records-per-page")
	viewOption := req.FormValue("view-option")
	pools := strings.Split(req.FormValue("pools"), "|")

	if viewOption == "" {
		viewOption = web.DefaultViewOption
	}

	if selectedDataType == "" {
		selectedDataType = "hashrate"
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

	if selectedPow == "" {
		selectedPow = "All"
	}

	offset := (pageToLoad - 1) * pageSize

	ctx := req.Context()

	data := map[string]interface{}{
		"chartView":          true,
		"selectedViewOption": viewOption,
		"selectedFilter":     selectedPow,
		"selectedDataType":   selectedDataType,
		"selectedPools":      pools,
		"pageSizeSelector":   web.PageSizeSelector,
		"selectedNum":        pageSize,
		"currentPage":        pageToLoad,
		"previousPage":       pageToLoad - 1,
	}

	powSource, err := c.store.FetchPowSourceData(ctx)
	if err != nil {
		return nil, err
	}

	if len(powSource) == 0 {
		return nil, fmt.Errorf("No PoW source data. Try running dcrextdata then try again.")
	}

	data["powSource"] = powSource

	if viewOption == web.DefaultViewOption {
		return data, nil
	}

	var totalCount int64
	var allPowDataSlice []PowDataDto
	if selectedPow == "All" || selectedPow == "" {
		allPowDataSlice, totalCount, err = c.store.FetchPowData(ctx, offset, pageSize)
		if err != nil {
			return nil, err
		}
	} else {
		allPowDataSlice, totalCount, err = c.store.FetchPowDataBySource(ctx, selectedPow, offset, pageSize)
		if err != nil {
			return nil, err
		}
	}

	if len(allPowDataSlice) == 0 {
		data["message"] = fmt.Sprintf("%s %s", strings.Title(selectedPow), web.NoDataMessage)
		return data, nil
	}

	data["powData"] = allPowDataSlice
	data["totalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))

	totalTxLoaded := offset + len(allPowDataSlice)
	if int64(totalTxLoaded) < totalCount {
		data["nextPage"] = pageToLoad + 1
	}

	return data, nil
}

// /api/charts/pow/{dataType}
func (c *Collector) chart(w http.ResponseWriter, r *http.Request) {
	dataType := web.GetChartDataTypeCtx(r)
	bin := r.URL.Query().Get("bin")

	chartData, err := c.store.FetchEncodePowChart(r.Context(), dataType, bin)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		log.Warnf(`Error fetching mempool %s chart: %v`, dataType, err)
		return
	}
	web.RenderJSONBytes(w, chartData)
}
