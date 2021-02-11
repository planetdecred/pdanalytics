package propagation

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/planetdecred/pdanalytics/web"
)

var (
	propagationRecordSet = map[string]string{
		"blocks": "Blocks",
		"votes":  "Votes",
	}
)

// propagationPage the handle the HTTP request to the /propagation endpoint
func (prop *propagation) propagationPage(w http.ResponseWriter, r *http.Request) {

	block, err := prop.fetchPropagationData(r)
	if err != nil {
		log.Error(err)
		prop.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	str, err := prop.server.Templates.ExecTemplateToString("propagation", struct {
		*web.CommonPageData
		Propagation map[string]interface{}
		BlockTime   float64
	}{
		CommonPageData: prop.server.CommonData(r),
		Propagation:    block,
		BlockTime:      prop.client.Params.MinDiffReductionTime.Seconds(),
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		prop.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

func (prop *propagation) getPropagationData(w http.ResponseWriter, r *http.Request) {
	data, err := prop.fetchPropagationData(r)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		return
	}
	web.RenderJSON(w, data)
}

func (prop *propagation) fetchPropagationData(req *http.Request) (map[string]interface{}, error) {
	req.ParseForm()
	page := req.FormValue("page")
	numberOfRows := req.FormValue("records-per-page")
	viewOption := req.FormValue("view-option")
	recordSet := req.FormValue("record-set")
	chartType := req.FormValue("chart-type")

	if viewOption == "" {
		viewOption = "chart"
	}

	if recordSet == "" {
		recordSet = "both"
	}

	if chartType == "" {
		chartType = "block-propagation"
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

	offset := (pageToLoad - 1) * pageSize

	ctx := req.Context()

	data := map[string]interface{}{
		"chartView":            viewOption == "chart",
		"selectedViewOption":   viewOption,
		"chartType":            chartType,
		"currentPage":          pageToLoad,
		"propagationRecordSet": propagationRecordSet,
		"pageSizeSelector":     web.PageSizeSelector,
		"selectedRecordSet":    recordSet,
		"both":                 true,
		"selectedNum":          pageSize,
		"url":                  "/propagation",
		"previousPage":         pageToLoad - 1,
		"totalPages":           0,
		"syncSources":          strings.Join(prop.externalDBs, "|"),
	}

	if viewOption == web.DefaultViewOption {
		return data, nil
	}

	blockSlice, err := prop.dataStore.BlocksWithoutVotes(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	for i := 0; i <= 1 && i <= len(blockSlice)-1; i++ {
		votes, err := prop.dataStore.VotesByBlock(ctx, blockSlice[i].BlockHash)
		if err != nil {
			return nil, err
		}
		blockSlice[i].Votes = votes
	}

	totalCount, err := prop.dataStore.BlockCount(ctx)
	if err != nil {
		return nil, err
	}

	if len(blockSlice) == 0 {
		data["message"] = fmt.Sprintf("%s %s", recordSet, web.NoDataMessage)
		return data, nil
	}

	data["records"] = blockSlice
	data["totalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))

	totalTxLoaded := offset + len(blockSlice)
	if int64(totalTxLoaded) < totalCount {
		data["nextPage"] = pageToLoad + 1
	}

	return data, nil
}

// getblocks handles the HTTP request to the /getblocks endpoint
func (c *propagation) getBlocks(res http.ResponseWriter, req *http.Request) {
	data, err := c.fetchBlockData(req)
	if err != nil {
		web.RenderErrorfJSON(res, err.Error())
		return
	}

	web.RenderJSON(res, data)
}

func (c *propagation) fetchBlockData(req *http.Request) (map[string]interface{}, error) {
	req.ParseForm()
	page := req.FormValue("page")
	numberOfRows := req.FormValue("records-per-page")
	viewOption := req.FormValue("view-option")

	if viewOption == "" {
		viewOption = web.DefaultViewOption
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

	offset := (pageToLoad - 1) * pageSize

	ctx := req.Context()

	data := map[string]interface{}{
		"chartView":            true,
		"selectedViewOption":   web.DefaultViewOption,
		"currentPage":          pageToLoad,
		"propagationRecordSet": propagationRecordSet,
		"pageSizeSelector":     web.PageSizeSelector,
		"selectedFilter":       "blocks",
		"blocks":               true,
		"url":                  "/blockdata",
		"selectedNum":          pageSize,
		"previousPage":         pageToLoad - 1,
		"totalPages":           pageToLoad,
	}

	if viewOption == web.DefaultViewOption {
		return data, nil
	}

	blocksSlice, err := c.dataStore.BlocksWithoutVotes(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	if len(blocksSlice) == 0 {
		data["message"] = fmt.Sprintf("Blocks %s", web.NoDataMessage)
		return data, nil
	}

	totalCount, err := c.dataStore.BlockCount(ctx)
	if err != nil {
		return nil, err
	}

	data["records"] = blocksSlice
	data["totalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))

	totalTxLoaded := offset + len(blocksSlice)
	if int64(totalTxLoaded) < totalCount {
		data["nextPage"] = pageToLoad + 1
	}

	return data, nil
}

// getvotes handles the HTTP request to the /getvotes endpoint
func (c *propagation) getVotes(res http.ResponseWriter, req *http.Request) {
	data, err := c.fetchVoteData(req)

	if err != nil {
		web.RenderErrorfJSON(res, err.Error())
		return
	}
	web.RenderJSON(res, data)
}

func (c *propagation) fetchVoteData(req *http.Request) (map[string]interface{}, error) {
	req.ParseForm()
	page := req.FormValue("page")
	numberOfRows := req.FormValue("records-per-page")
	viewOption := req.FormValue("view-option")

	if viewOption == "" {
		viewOption = web.DefaultViewOption
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

	offset := (pageToLoad - 1) * pageSize

	ctx := req.Context()

	data := map[string]interface{}{
		"chartView":            true,
		"selectedViewOption":   web.DefaultViewOption,
		"currentPage":          pageToLoad,
		"propagationRecordSet": propagationRecordSet,
		"pageSizeSelector":     web.PageSizeSelector,
		"selectedFilter":       "votes",
		"votes":                true,
		"selectedNum":          pageSize,
		"url":                  "/votesdata",
		"previousPage":         pageToLoad - 1,
		"totalPages":           pageToLoad,
	}

	if viewOption == web.DefaultViewOption {
		return data, nil
	}

	voteSlice, err := c.dataStore.Votes(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	if len(voteSlice) == 0 {
		data["message"] = fmt.Sprintf("Votes %s", web.NoDataMessage)
		return data, nil
	}

	totalCount, err := c.dataStore.VotesCount(ctx)
	if err != nil {
		return nil, err
	}

	data["voteRecords"] = voteSlice
	data["totalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))

	totalTxLoaded := offset + len(voteSlice)
	if int64(totalTxLoaded) < totalCount {
		data["nextPage"] = pageToLoad + 1
	}

	return data, nil
}

// getVoteByBlock handles the HTTP request to the /getVoteByBlock endpoint
func (c *propagation) getVoteByBlock(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	hash := req.FormValue("block_hash")
	votes, err := c.dataStore.VotesByBlock(req.Context(), hash)
	if err != nil {
		web.RenderErrorfJSON(res, err.Error())
		return
	}
	web.RenderJSON(res, votes)
}

// api/charts/propagation/{dataType}
func (c *propagation) chart(w http.ResponseWriter, r *http.Request) {
	dataType := getChartDataTypeCtx(r)
	bin := r.URL.Query().Get("bin")
	axis := r.URL.Query().Get("axis")
	extras := r.URL.Query().Get("extras")

	chartData, err := c.dataStore.FetchEncodePropagationChart(r.Context(), dataType, axis, bin, extras)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		log.Warnf(`Error fetching mempool %s chart: %v`, dataType, err)
		return
	}
	web.RenderJSONBytes(w, chartData)
}

// chartDataTypeCtx returns a http.HandlerFunc that embeds the value at the url
// part {chartAxisType} into the request context.
func chartDataTypeCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "ctxChartDataType",
			chi.URLParam(r, "chartDataType"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getChartDataTypeCtx retrieves the ctxChartAxisType data from the request context.
// If not set, the return value is an empty string.
func getChartDataTypeCtx(r *http.Request) string {
	chartAxisType, ok := r.Context().Value("ctxChartDataType").(string)
	if !ok {
		log.Trace("chart axis type not set")
		return ""
	}
	return chartAxisType
}
