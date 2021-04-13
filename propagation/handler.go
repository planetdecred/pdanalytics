package propagation

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/planetdecred/pdanalytics/chart"
	"github.com/planetdecred/pdanalytics/web"
)

const (
	// chart data types
	BlockPropagation = "block-propagation"
	BlockTimestamp   = "block-timestamp"
	VotesReceiveTime = "votes-receive-time"
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
		Propagation     map[string]interface{}
		BlockTime       float64
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: prop.server.CommonData(r),
		Propagation:    block,
		BlockTime:      prop.client.Params.MinDiffReductionTime.Seconds(),
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Block Propagation",
				Active:    true,
			},
		},
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
		"syncSources":          strings.Join(prop.externalDBNames, "|"),
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
func (prop *propagation) getBlocks(res http.ResponseWriter, req *http.Request) {
	data, err := prop.fetchBlockData(req)
	if err != nil {
		web.RenderErrorfJSON(res, err.Error())
		return
	}

	web.RenderJSON(res, data)
}

func (prop *propagation) fetchBlockData(req *http.Request) (map[string]interface{}, error) {
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

	blocksSlice, err := prop.dataStore.BlocksWithoutVotes(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	if len(blocksSlice) == 0 {
		data["message"] = fmt.Sprintf("Blocks %s", web.NoDataMessage)
		return data, nil
	}

	totalCount, err := prop.dataStore.BlockCount(ctx)
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
func (prop *propagation) getVotes(res http.ResponseWriter, req *http.Request) {
	data, err := prop.fetchVoteData(req)

	if err != nil {
		web.RenderErrorfJSON(res, err.Error())
		return
	}
	web.RenderJSON(res, data)
}

func (prop *propagation) fetchVoteData(req *http.Request) (map[string]interface{}, error) {
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

	voteSlice, err := prop.dataStore.Votes(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}

	if len(voteSlice) == 0 {
		data["message"] = fmt.Sprintf("Votes %s", web.NoDataMessage)
		return data, nil
	}

	totalCount, err := prop.dataStore.VotesCount(ctx)
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
func (prop *propagation) getVoteByBlock(res http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	hash := req.FormValue("block_hash")
	votes, err := prop.dataStore.VotesByBlock(req.Context(), hash)
	if err != nil {
		web.RenderErrorfJSON(res, err.Error())
		return
	}
	web.RenderJSON(res, votes)
}

// chart handles api/charts/propagation/{dataType} endpoint.
// dataType is one of block, vote, block-propagation
func (prop *propagation) chart(w http.ResponseWriter, r *http.Request) {
	dataType := getChartDataTypeCtx(r)
	bin := r.URL.Query().Get("bin")
	axis := r.URL.Query().Get("axis")

	chartData, err := prop.fetchEncodePropagationChart(r.Context(), dataType, axis, bin)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		log.Warnf(`Error fetching mempool %s chart: %v`, dataType, err)
		return
	}
	web.RenderJSONBytes(w, chartData)
}

func (prop *propagation) fetchEncodePropagationChart(ctx context.Context, dataType, axis string,
	binString string) ([]byte, error) {

	switch dataType {
	case BlockPropagation:
		return prop.blockPropagationChart(ctx, axis, binString)

	case BlockTimestamp:
		return prop.blockTimestampChart(ctx, axis, binString)

	case VotesReceiveTime:
		return prop.votesReceiveTimeChart(ctx, axis, binString)
	}
	return nil, chart.UnknownChartErr
}

func (prop *propagation) blockPropagationChart(ctx context.Context, axis string, binString string) ([]byte, error) {
	blockPropagation := make(map[string]chart.ChartFloats)
	var dates chart.ChartUints
	dateMap := make(map[int64]bool)
	for _, source := range prop.externalDBNames {
		data, err := prop.dataStore.SourceDeviations(ctx, source, binString)
		if err != nil {
			return nil, err
		}

		for _, rec := range data {
			if _, f := dateMap[rec.Height]; !f {
				if axis == string(chart.HeightAxis) {
					dates = append(dates, uint64(rec.Height))
				} else {
					dates = append(dates, uint64(rec.Time))
				}
				dateMap[rec.Height] = true
			}
			blockPropagation[source] = append(blockPropagation[source], rec.Deviation)
		}
	}
	var data = []chart.Lengther{dates}
	for _, d := range blockPropagation {
		data = append(data, d)
	}
	return chart.Encode(nil, data...)
}

func (prop *propagation) blockTimestampChart(ctx context.Context, axis string, binString string) ([]byte, error) {
	if binString == string(chart.DefaultBin) {
		blockDelays, err := prop.dataStore.BlockDelays(ctx, 0)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}

		var xAxis chart.ChartUints
		var blockDelay chart.ChartFloats
		localBlockReceiveTime := make(map[uint64]float64)
		for _, record := range blockDelays {
			if axis == string(chart.HeightAxis) {
				xAxis = append(xAxis, uint64(record.BlockHeight))
			} else {
				xAxis = append(xAxis, uint64(record.BlockTime.Unix()))
			}
			timeDifference, _ := strconv.ParseFloat(fmt.Sprintf("%04.2f", record.TimeDifference), 64)
			blockDelay = append(blockDelay, timeDifference)

			localBlockReceiveTime[uint64(record.BlockHeight)] = timeDifference
		}
		return chart.Encode(nil, xAxis, blockDelay)
	} else {
		blocks, err := prop.dataStore.BlockBinData(ctx, binString)
		if err != nil {
			return nil, err
		}
		var xAxis chart.ChartUints
		var blockDelay chart.ChartFloats
		for _, block := range blocks {
			if axis == string(chart.HeightAxis) {
				xAxis = append(xAxis, uint64(block.Height))
			} else {
				xAxis = append(xAxis, uint64(block.InternalTimestamp))
			}
			blockDelay = append(blockDelay, block.ReceiveTimeDiff)
		}
		return chart.Encode(nil, xAxis, blockDelay)
	}
}

func (prop *propagation) votesReceiveTimeChart(ctx context.Context, axis string, binString string) ([]byte, error) {
	blockDelays, err := prop.dataStore.BlockDelays(ctx, 0)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	var xAxis chart.ChartUints
	for _, record := range blockDelays {
		if axis == string(chart.HeightAxis) {
			xAxis = append(xAxis, uint64(record.BlockHeight))
		} else {
			xAxis = append(xAxis, uint64(record.BlockTime.Unix()))
		}
	}

	if binString == string(chart.DefaultBin) {
		votesReceiveTime, err := prop.dataStore.VotesBlockReceiveTimeDiffs(ctx)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		var votesTimeDeviations = make(map[int64]chart.ChartFloats)

		for _, record := range votesReceiveTime {
			votesTimeDeviations[record.BlockHeight] = append(votesTimeDeviations[record.BlockHeight], record.TimeDifference)
		}

		var voteReceiveTimeDeviations chart.ChartFloats
		for _, height := range xAxis {
			if deviations, found := votesTimeDeviations[int64(height)]; found {
				var totalTime float64
				for _, timeDiff := range deviations {
					totalTime += timeDiff
				}
				timeDifference, _ := strconv.ParseFloat(fmt.Sprintf("%04.2f", totalTime/float64(len(deviations))*1000), 64)
				voteReceiveTimeDeviations = append(voteReceiveTimeDeviations, timeDifference)
				continue
			}
			voteReceiveTimeDeviations = append(voteReceiveTimeDeviations, 0)
		}
		return chart.Encode(nil, xAxis, voteReceiveTimeDeviations)
	} else {
		records, err := prop.dataStore.VoteReceiveTimeDeviations(ctx, binString)
		if err != nil {
			return nil, err
		}
		var xAxis chart.ChartUints
		var diffs chart.ChartFloats
		for _, rec := range records {
			if axis == string(chart.HeightAxis) {
				xAxis = append(xAxis, uint64(rec.BlockHeight))
			} else {
				xAxis = append(xAxis, uint64(rec.BlockTime))
			}
			diffs = append(diffs, rec.ReceiveTimeDifference)
		}
		return chart.Encode(nil, xAxis, diffs)
	}
}

// chartDataTypeCtx returns a http.HandlerFunc that embeds the value at the url
// part {chartAxisType} into the request context.
func chartDataTypeCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), web.CtxChartDataType,
			chi.URLParam(r, "chartDataType"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getChartDataTypeCtx retrieves the ctxChartAxisType data from the request context.
// If not set, the return value is an empty string.
func getChartDataTypeCtx(r *http.Request) string {
	chartAxisType, ok := r.Context().Value(web.CtxChartDataType).(string)
	if !ok {
		log.Trace("chart axis type not set")
		return ""
	}
	return chartAxisType
}
