package netsnapshot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/planetdecred/pdanalytics/chart"
	"github.com/planetdecred/pdanalytics/web"
)

const (
	SnapshotNodes          = "nodes"
	SnapshotReachableNodes = "reachable-nodes"
	SnapshotLocations      = "locations"
	SnapshotNodeVersions   = "node-versions"
)

// nodesPage handes http request to /nodes endpoint
func (t *taker) nodesPage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	page, _ := strconv.Atoi(r.FormValue("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.FormValue("page-size"))
	if pageSize < 1 {
		pageSize = web.DefaultPageSize
	}

	viewOption := r.FormValue("view-option")
	if viewOption == "" {
		viewOption = web.DefaultViewOption
	}

	var timestamp, previousTimestamp, nextTimestamp int64

	tstamp, _ := strconv.Atoi(r.FormValue("timestamp"))
	timestamp = int64(tstamp)

	if timestamp == 0 {
		timestamp = t.dataStore.LastSnapshotTime(r.Context())
		if timestamp == 0 {
			msg := "No snapshot has been taken, please confirm that snapshot taker is configured."
			t.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, msg, web.ExpStatusError)
			return
		}
	}

	if snapshot, err := t.dataStore.PreviousSnapshot(r.Context(), timestamp); err == nil {
		previousTimestamp = snapshot.Timestamp
	}

	if snapshot, err := t.dataStore.NextSnapshot(r.Context(), timestamp); err == nil {
		nextTimestamp = snapshot.Timestamp
	}

	snapshot, err := t.dataStore.FindNetworkSnapshot(r.Context(), timestamp)
	if err != nil {
		msg := fmt.Sprintf("Cannot find a snapshot of the specified timestamp, %s", err.Error())
		t.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, msg, web.ExpStatusError)
		return
	}

	dataType := r.FormValue("data-type")
	if dataType == "" {
		dataType = "nodes"
	}

	//
	var totalCount, pageCount int64
	switch dataType {
	case "snapshot":
	default:
		totalCount, err = t.dataStore.SnapshotCount(r.Context())
		if err != nil {
			t.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, err.Error(), web.ExpStatusError)
			return
		}
	}

	if totalCount%int64(pageSize) == 0 {
		pageCount = totalCount / int64(pageSize)
	} else {
		pageCount = 1 + (totalCount-totalCount%int64(pageSize))/int64(pageSize)
	}

	var previousPage int = page - 1
	var nextPage int = page + 1

	data := map[string]interface{}{
		"selectedViewOption": viewOption,
		"dataType":           dataType,
		"pageSizeSelector":   web.PageSizeSelector,
		"previousPage":       previousPage,
		"currentPage":        page,
		"nextPage":           nextPage,
		"pageSize":           pageSize,
		"totalPages":         pageCount,
		"timestamp":          timestamp,
		"height":             snapshot.Height,
		"previousTimestamp":  previousTimestamp,
		"nextTimestamp":      nextTimestamp,
	}

	str, err := t.server.Templates.ExecTemplateToString("nodes", struct {
		*web.CommonPageData
		Data map[string]interface{}
	}{
		CommonPageData: t.server.CommonData(r),
		Data:           data,
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		t.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

// /api/snapshots
func (t *taker) snapshots(w http.ResponseWriter, r *http.Request) {
	pageSize, err := strconv.Atoi(r.FormValue("page-size"))
	if err != nil {
		pageSize = web.DefaultPageSize
	}

	page, err := strconv.Atoi(r.FormValue("page"))
	if err != nil {
		page = 1
	}

	offset := (page - 1) * pageSize

	result, total, err := t.dataStore.Snapshots(r.Context(), offset, pageSize, false)
	if err != nil {
		web.RenderErrorfJSON(w, "Cannot fetch snapshots: %s", err.Error())
		return
	}
	var totalPages int64
	if total%int64(pageSize) == 0 {
		totalPages = total / int64(pageSize)
	} else {
		totalPages = 1 + (total-total%int64(pageSize))/int64(pageSize)
	}
	web.RenderJSON(w, map[string]interface{}{"data": result, "total": total, "totalPages": totalPages})
}

// /api/snapshots/chart
func (t *taker) snapshotsChart(w http.ResponseWriter, r *http.Request) {
	result, _, err := t.dataStore.Snapshots(r.Context(), 0, -1, true)
	if err != nil {
		web.RenderErrorfJSON(w, "Cannot fetch snapshots: %s", err.Error())
		return
	}
	web.RenderJSON(w, result)
}

// /api/snapshots/user-agents
func (t *taker) nodesCountUserAgents(w http.ResponseWriter, r *http.Request) {
	pageSize, err := strconv.Atoi(r.FormValue("page-size"))
	if err != nil {
		pageSize = web.DefaultPageSize
	}

	page, _ := strconv.Atoi(r.FormValue("page"))
	var offset int
	if page < 1 {
		page = 1
	}
	offset = (page - 1) * pageSize

	userAgents, total, err := t.dataStore.FetchNodeVersion(r.Context(), offset, pageSize)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		return
	}

	var totalPages int64
	if total%int64(pageSize) == 0 {
		totalPages = total / int64(pageSize)
	} else {
		totalPages = 1 + (total-total%int64(pageSize))/int64(pageSize)
	}
	web.RenderJSON(w, map[string]interface{}{"userAgents": userAgents, "totalPages": totalPages})
}

// /api/snapshots/user-agents/chart
func (t *taker) nodesCountUserAgentsChart(w http.ResponseWriter, r *http.Request) {
	limit := -1
	offset := 0
	var err error
	userAgents := []UserAgentInfo{}
	sources := r.FormValue("sources")
	if len(sources) > 0 {
		userAgents, _, err = t.dataStore.PeerCountByUserAgents(r.Context(), sources, offset, limit)
		if err != nil {
			web.RenderErrorfJSON(w, "Cannot fetch data: %s", err.Error())
			return
		}
	}

	var datesMap = map[int64]struct{}{}
	var allDates []int64
	var userAgentMap = map[string]struct{}{}
	var allUserAgents []string
	var dateUserAgentCount = make(map[int64]map[string]int64)

	for _, item := range userAgents {
		if _, exists := datesMap[item.Timestamp]; !exists {
			datesMap[item.Timestamp] = struct{}{}
			allDates = append(allDates, item.Timestamp)
		}

		if _, exists := dateUserAgentCount[item.Timestamp]; !exists {
			dateUserAgentCount[item.Timestamp] = make(map[string]int64)
		}

		if _, exists := userAgentMap[item.UserAgent]; !exists {
			userAgentMap[item.UserAgent] = struct{}{}
			allUserAgents = append(allUserAgents, item.UserAgent)
		}
		dateUserAgentCount[item.Timestamp][item.UserAgent] = item.Nodes
	}

	var row = []string{"Date (UTC)"}
	row = append(row, allUserAgents...)
	csv := strings.Join(row, ",") + "\n"

	var minDate, maxDate int64
	for _, timestamp := range allDates {
		if minDate == 0 || timestamp < minDate {
			minDate = timestamp
		}

		if maxDate == 0 || timestamp > maxDate {
			maxDate = timestamp
		}

		row = []string{time.Unix(timestamp, 0).UTC().String()}
		for _, userAgent := range allUserAgents {
			row = append(row, strconv.FormatInt(dateUserAgentCount[timestamp][userAgent], 10))
		}
		csv += strings.Join(row, ",") + "\n"
	}

	web.RenderJSON(w, map[string]interface{}{
		"csv":     csv,
		"minDate": time.Unix(minDate, 0).UTC().String(),
		"maxDate": time.Unix(maxDate, 0).UTC().String(),
	})
}

// /api/snapshots/countries
func (t *taker) nodesCountByCountries(w http.ResponseWriter, r *http.Request) {
	pageSize, err := strconv.Atoi(r.FormValue("page-size"))
	if err != nil {
		pageSize = web.DefaultPageSize
	}

	page, _ := strconv.Atoi(r.FormValue("page"))
	var offset int
	if page < 1 {
		page = 1
	}
	offset = (page - 1) * pageSize

	countries, total, err := t.dataStore.FetchNodeLocations(r.Context(), offset, pageSize)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error(), w)
		return
	}

	var totalPages int64
	if total%int64(pageSize) == 0 {
		totalPages = total / int64(pageSize)
	} else {
		totalPages = 1 + (total-total%int64(pageSize))/int64(pageSize)
	}

	web.RenderJSON(w, map[string]interface{}{"countries": countries, "totalPages": totalPages})
}

// /api/snapshots/countries/chart
func (t *taker) nodesCountByCountriesChart(w http.ResponseWriter, r *http.Request) {
	limit := -1
	offset := 0
	sources := r.FormValue("sources")
	var err error
	countries := []CountryInfo{}
	if len(sources) > 0 {
		countries, _, err = t.dataStore.PeerCountByCountries(r.Context(), sources, offset, limit)
		if err != nil {
			web.RenderErrorfJSON(w, "Cannot fetch data: %s", err.Error())
			return
		}
	}

	var datesMap = map[int64]struct{}{}
	var allDates []int64
	var countryMap = map[string]struct{}{}
	var allCountries []string
	var dateCountryCount = make(map[int64]map[string]int64)

	for _, item := range countries {
		if _, exists := datesMap[item.Timestamp]; !exists {
			datesMap[item.Timestamp] = struct{}{}
			allDates = append(allDates, item.Timestamp)
		}

		if _, exists := dateCountryCount[item.Timestamp]; !exists {
			dateCountryCount[item.Timestamp] = make(map[string]int64)
		}

		if _, exists := countryMap[item.Country]; !exists {
			countryMap[item.Country] = struct{}{}
			allCountries = append(allCountries, item.Country)
		}
		dateCountryCount[item.Timestamp][item.Country] = item.Nodes
	}

	var row = []string{"Date (UTC)"}
	row = append(row, allCountries...)
	csv := strings.Join(row, ",") + "\n"

	var minDate, maxDate int64
	for _, timestamp := range allDates {
		if minDate == 0 || timestamp < minDate {
			minDate = timestamp
		}

		if maxDate == 0 || timestamp > maxDate {
			maxDate = timestamp
		}

		row = []string{time.Unix(timestamp, 0).UTC().String()}
		for _, country := range allCountries {
			row = append(row, strconv.FormatInt(dateCountryCount[timestamp][country], 10))
		}
		csv += strings.Join(row, ",") + "\n"
	}

	web.RenderJSON(w, map[string]interface{}{
		"csv":     csv,
		"minDate": time.Unix(minDate, 0).UTC().String(),
		"maxDate": time.Unix(maxDate, 0).UTC().String(),
	})
}

// /api/snapshot/nodes/count-by-timestamp
func (t *taker) nodeCountByTimestamp(w http.ResponseWriter, r *http.Request) {
	result, err := t.dataStore.SeenNodesByTimestamp(r.Context())
	if err != nil {
		web.RenderErrorfJSON(w, "Cannot fetch node count: %s", err.Error())
		return
	}
	web.RenderJSON(w, result)
}

func getTitmestampCtx(r *http.Request) int64 {
	timestampStr, ok := r.Context().Value(web.CtxTimestamp).(string)
	if !ok {
		return 0
	}
	timestamp, _ := strconv.ParseInt(timestampStr, 10, 64)
	return timestamp
}

func addTimestampToCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), web.CtxTimestamp,
			chi.URLParam(r, "timestamp"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// /api/snapshot/{timestamp}/nodes
func (t *taker) nodes(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	page, _ := strconv.Atoi(r.FormValue("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.FormValue("page-size"))
	if pageSize < 1 {
		pageSize = web.DefaultPageSize
	}

	offset := (page - 1) * pageSize
	query := r.FormValue("q")

	timestamp := getTitmestampCtx(r)
	if timestamp == 0 {
		web.RenderErrorfJSON(w, "timestamp is required and cannot be zero")
		return
	}

	nodes, peerCount, err := t.dataStore.NetworkPeers(r.Context(), timestamp, query, offset, pageSize)
	if err != nil {
		web.RenderErrorfJSON(w, "Error in fetching network nodes, %s", err.Error())
		return
	}

	rem := peerCount % web.DefaultPageSize
	pageCount := (peerCount - rem) / web.DefaultPageSize
	if rem > 0 {
		pageCount += 1
	}

	web.RenderJSON(w, map[string]interface{}{
		"page":      page,
		"pageCount": pageCount,
		"peerCount": peerCount,
		"nodes":     nodes,
	})
}

// /api/snapshots/ip-info
func (t *taker) ipInfo(w http.ResponseWriter, r *http.Request) {
	address := r.FormValue("ip")
	if address == "" {
		web.RenderErrorfJSON(w, "please specify a valid IP")
		return
	}
	country, version, err := t.dataStore.GetIPLocation(r.Context(), address)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		return
	}

	web.RenderJSON(w, map[string]interface{}{"country": country, "ip_version": version})
}

// api/snapshot/node-versions
func (t *taker) nodeVersions(w http.ResponseWriter, r *http.Request) {
	version, err := t.dataStore.AllNodeVersions(r.Context())
	if err != nil {
		web.RenderErrorfJSON(w, "Cannot fetch node versions - %s", err.Error())
		return
	}
	web.RenderJSON(w, version)
}

// api/snapshot/node-countries
func (t *taker) nodeCountries(w http.ResponseWriter, r *http.Request) {
	version, err := t.dataStore.AllNodeContries(r.Context())
	if err != nil {
		web.RenderErrorfJSON(w, "Cannot fetch node contries - %s", err.Error())
		return
	}
	web.RenderJSON(w, version)
}

// chart processing

// api/charts/snapshot/{dataType}
func (t *taker) chart(w http.ResponseWriter, r *http.Request) {
	dataType := web.GetChartDataTypeCtx(r)
	bin := r.URL.Query().Get("bin")
	axis := r.URL.Query().Get("axis")
	extras := strings.Split(r.URL.Query().Get("extras"), "|")

	chartData, err := t.fetchEncodeSnapshotChart(r.Context(), dataType, axis, bin, extras...)
	if err != nil {
		web.RenderErrorfJSON(w, err.Error())
		log.Warnf(`Error fetching mempool %s chart: %v`, dataType, err)
		return
	}
	web.RenderJSONBytes(w, chartData)
}

func (t *taker) fetchEncodeSnapshotChart(ctx context.Context, dataType, axis, binString string, extras ...string) ([]byte, error) {
	switch dataType {
	case SnapshotNodes:
		return t.fetchEncodeSnapshotNodesChart(ctx, dataType, binString)
	case SnapshotNodeVersions:
		return t.fetchEncodeSnapshotNodeVersionsChart(ctx, axis, binString, extras...)
	case SnapshotLocations:
		return t.fetchEncodeSnapshotLocationsChart(ctx, axis, binString, extras...)
	default:
		return nil, chart.UnknownChartErr
	}
}

func (t *taker) fetchEncodeSnapshotNodesChart(ctx context.Context, dataType, binString string) ([]byte, error) {
	var time, nodes, reachableNodes chart.ChartUints

	if binString == string(chart.DefaultBin) {
		result, err := t.dataStore.SnapshotsByTime(ctx, 0, 0)
		if err != nil {
			return nil, err
		}

		for _, rec := range result {
			if dataType == string(chart.HeightAxis) {
				time = append(time, uint64(rec.Height))
			} else {
				time = append(time, uint64(rec.Timestamp))
			}
			nodes = append(nodes, uint64(rec.NodeCount))
			reachableNodes = append(reachableNodes, uint64(rec.ReachableNodeCount))
		}
	} else {
		result, err := t.dataStore.SnapshotsByBin(ctx, binString)
		if err != nil {
			return nil, err
		}

		for _, rec := range result {
			if dataType == string(chart.HeightAxis) {
				time = append(time, uint64(rec.Height))
			} else {
				time = append(time, uint64(rec.Timestamp))
			}
			nodes = append(nodes, uint64(rec.NodeCount))
			reachableNodes = append(reachableNodes, uint64(rec.ReachableNodeCount))
		}
	}

	return chart.Encode(nil, time, nodes, reachableNodes)
}

func (t *taker) fetchEncodeSnapshotNodeVersionsChart(ctx context.Context, axis, binString string, userAgentsArg ...string) ([]byte, error) {
	datesMap := map[int64]struct{}{}
	allDates := chart.ChartUints{}
	versions := map[string]chart.ChartUints{}
	for _, userAgent := range userAgentsArg {
		records, err := t.dataStore.NodeVersionsByBin(ctx, userAgent, binString)
		if err != nil {
			return nil, err
		}
		var nodeCounts chart.ChartUints
		for _, rec := range records {
			if _, f := datesMap[rec.Timestamp]; !f {
				if axis == string(chart.HeightAxis) {
					allDates = append(allDates, uint64(rec.Height))
				} else {
					allDates = append(allDates, uint64(rec.Timestamp))
				}
			}
			nodeCounts = append(nodeCounts, uint64(rec.Nodes))
		}
		versions[userAgent] = nodeCounts
	}

	recs := []chart.Lengther{allDates}

	for _, v := range userAgentsArg {
		recs = append(recs, versions[v])
	}

	return chart.Encode(nil, recs...)
}

func (t *taker) fetchEncodeSnapshotLocationsChart(ctx context.Context, axis, binString string, countriesArg ...string) ([]byte, error) {
	var datesMap = map[int64]struct{}{}
	var allDates chart.ChartUints

	locationSet := map[string]chart.ChartUints{}
	for _, country := range countriesArg {
		records, err := t.dataStore.NodeLocationsByBin(ctx, country, binString)
		if err != nil {
			return nil, err
		}
		spew.Dump(records, country)
		var nodeCounts chart.ChartUints
		for _, rec := range records {
			if _, f := datesMap[rec.Timestamp]; !f {
				if axis == string(chart.HeightAxis) {
					allDates = append(allDates, uint64(rec.Height))
				} else {
					allDates = append(allDates, uint64(rec.Timestamp))
				}
			}
			nodeCounts = append(nodeCounts, uint64(rec.Nodes))
		}
		locationSet[country] = nodeCounts
	}

	recs := []chart.Lengther{allDates}
	for _, c := range countriesArg {
		recs = append(recs, locationSet[c])
	}

	return chart.Encode(nil, recs...)
}
