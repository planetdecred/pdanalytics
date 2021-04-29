package vsp

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/planetdecred/pdanalytics/web"
)

var (
	allVspDataTypes = []string{
		"Immature",
		"Live",
		"Voted",
		"Missed",
		"Pool-Fees",
		"Proportion-Live",
		"Proportion-Missed",
		"User-Count",
		"Users-Active",
	}
)

// /vsps
func (vsp *Collector) vspPage(w http.ResponseWriter, r *http.Request) {
	data, err := vsp.fetchVSPData(r)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		return
	}

	str, err := vsp.server.Templates.ExecTemplateToString("vsp", struct {
		*web.CommonPageData
		BreadcrumbItems []web.BreadcrumbItem
		Data            map[string]interface{}
	}{
		CommonPageData: vsp.server.CommonData(r),
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Voting Service Provider data",
				Active:    true,
			},
		},
		Data: data,
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		vsp.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, err.Error(), web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

func (vsp *Collector) getFilteredVspTicks(w http.ResponseWriter, r *http.Request) {
	data, err := vsp.fetchVSPData(r)

	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		return
	}

	web.RenderJSON(w, data)
}

func (vsp *Collector) fetchVSPData(req *http.Request) (map[string]interface{}, error) {
	req.ParseForm()
	page := req.FormValue("page")
	selectedVsp := req.FormValue("filter")
	numberOfRows := req.FormValue("records-per-page")
	viewOption := req.FormValue("view-option")
	dataType := req.FormValue("data-type")
	selectedVsps := strings.Split(req.FormValue("vsps"), "|")

	if viewOption == "" {
		viewOption = web.DefaultViewOption
	}

	if dataType == "" {
		dataType = "Immature"
	}

	var pageSize int
	numRows, err := strconv.Atoi(numberOfRows)
	if err != nil || numRows <= 0 {
		pageSize = web.DefaultPageSize
	} else if numRows > web.MaxPageSize {
		pageSize = web.MaxPageSize
	} else {
		pageSize = numRows
	}

	pageToLoad, err := strconv.Atoi(page)
	if err != nil || pageToLoad <= 0 {
		pageToLoad = 1
	}

	if selectedVsp == "" {
		selectedVsp = "All"
	}

	offset := (pageToLoad - 1) * pageSize

	ctx := req.Context()

	data := map[string]interface{}{
		"chartView":          true,
		"selectedViewOption": viewOption,
		"selectedFilter":     selectedVsp,
		"pageSizeSelector":   web.PageSizeSelector,
		"selectedNum":        pageSize,
		"currentPage":        pageToLoad,
		"previousPage":       pageToLoad - 1,
		"totalPages":         0,
		"allDataTypes":       allVspDataTypes,
		"dataType":           dataType,
		"selectedVsps":       selectedVsps,
	}

	allVspData, err := vsp.dataStore.FetchVSPs(ctx)
	if err != nil {
		return nil, err
	}

	if len(allVspData) == 0 {
		return nil, fmt.Errorf("No VSP source data. Try running dcrextdata then try again.")
	}
	data["allVspData"] = allVspData

	if viewOption == "chart" {
		return data, nil
	}

	var allVSPSlice []VSPTickDto
	var totalCount int64
	if selectedVsp == "All" || selectedVsp == "" {
		allVSPSlice, totalCount, err = vsp.dataStore.AllVSPTicks(ctx, offset, pageSize)
		if err != nil {
			return nil, err
		}
	} else {
		allVSPSlice, totalCount, err = vsp.dataStore.FilteredVSPTicks(ctx, selectedVsp, offset, pageSize)
		if err != nil {
			return nil, err
		}
	}

	if len(allVSPSlice) == 0 {
		data["message"] = fmt.Sprintf("%s %s", strings.Title(selectedVsp), web.NoDataMessage)
		return data, nil
	}

	data["vspData"] = allVSPSlice
	data["allVspData"] = allVspData
	data["totalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))

	totalTxLoaded := offset + len(allVSPSlice)
	if int64(totalTxLoaded) < totalCount {
		data["nextPage"] = pageToLoad + 1
	}

	return data, nil
}

// /api/charts/pow/{dataType}
func (c *Collector) chart(w http.ResponseWriter, r *http.Request) {
	dataType := web.GetChartDataTypeCtx(r)
	bin := r.URL.Query().Get("bin")
	extras := r.URL.Query().Get("extras")
	sources := strings.Split(extras, "|")

	chartData, err := c.dataStore.FetchEncodeVspChart(r.Context(), dataType, bin, sources...)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		log.Warnf(`Error fetching mempool %s chart: %v`, dataType, err)
		return
	}
	web.RenderJSONBytes(w, chartData)
}
