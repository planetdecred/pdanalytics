package exchanges

import (
	"database/sql"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/planetdecred/pdanalytics/web"
)

var (
	noDataMessage = "does not have data for the selected query option(s)."

	exchangeTickIntervals = map[int]string{
		-1:   "All",
		5:    "5m",
		60:   "1h",
		120:  "2h",
		1440: "1d",
	}
)

func (s *TickHub) getExchangeTicks(w http.ResponseWriter, r *http.Request) {
	exchanges, err := s.fetchExchangeData(r)
	if err != nil {
		log.Errorf("fetchExchangeData execute failure: %v", err)
		s.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	str, err := s.server.Templates.ExecTemplateToString("exchange", struct {
		*web.CommonPageData
		BreadcrumbItems []web.BreadcrumbItem
		Data map[string]interface{}
	}{
		CommonPageData: s.server.CommonData(r),
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Historic exchange rate data",
				Active:    true,
			},
		},
		Data: exchanges,
	})

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

func (s *TickHub) getFilteredExchangeTicks(w http.ResponseWriter, r *http.Request) {
	data, err := s.fetchExchangeData(r)

	if err != nil {
		fmt.Println(err)
		web.RenderErrorfJSON(w, err.Error())
		return
	}

	web.RenderJSON(w, data)
}

func (s *TickHub) fetchExchangeData(req *http.Request) (map[string]interface{}, error) {
	req.ParseForm()
	page := req.FormValue("page")
	selectedExchange := req.FormValue("selected-exchange")
	numberOfRows := req.FormValue("records-per-page")
	selectedCurrencyPair := req.FormValue("selected-currency-pair")
	interval := req.FormValue("selected-interval")
	selectedTick := req.FormValue("selected-tick")
	viewOption := req.FormValue("view-option")

	if viewOption == "" {
		viewOption = web.DefaultViewOption
	}

	if selectedTick == "" {
		selectedTick = "close"
	}

	ctx := req.Context()

	var pageSize int
	numRows, err := strconv.Atoi(numberOfRows)
	if err != nil || numRows <= 0 {
		pageSize = web.DefaultPageSize
	} else if numRows > web.MaxPageSize {
		pageSize = web.MaxPageSize
	} else {
		pageSize = numRows
	}

	selectedInterval, err := strconv.Atoi(interval)
	if err != nil || selectedInterval <= 0 {
		selectedInterval = web.DefaultInterval
	}

	if _, found := exchangeTickIntervals[selectedInterval]; !found {
		selectedInterval = web.DefaultInterval
	}

	pageToLoad, err := strconv.Atoi(page)
	if err != nil || pageToLoad <= 0 {
		pageToLoad = 1
	}

	currencyPairs, err := s.store.AllExchangeTicksCurrencyPair(ctx)
	if err != nil {
		return nil, fmt.Errorf("Cannot fetch currency pair, %s", err.Error())
	}

	if selectedCurrencyPair == "" {
		if viewOption == "table" {
			selectedCurrencyPair = "All"
		} else if len(currencyPairs) > 0 {
			selectedCurrencyPair = currencyPairs[0].CurrencyPair
		}
	}

	offset := (pageToLoad - 1) * pageSize

	data := map[string]interface{}{
		"chartView":            true,
		"selectedViewOption":   viewOption,
		"intervals":            exchangeTickIntervals,
		"pageSizeSelector":     web.PageSizeSelector,
		"selectedCurrencyPair": selectedCurrencyPair,
		"selectedNum":          pageSize,
		"selectedInterval":     selectedInterval,
		"selectedTick":         selectedTick,
		"currentPage":          pageToLoad,
		"previousPage":         pageToLoad - 1,
		"totalPages":           0,
	}

	allExchangeSlice, err := s.store.AllExchange(ctx)
	if err != nil {
		return nil, fmt.Errorf("Cannot fetch exchanges, %s", err.Error())
	}

	if len(allExchangeSlice) == 0 {
		return nil, fmt.Errorf("No exchange source data. Try running dcrextdata then try again.")
	}
	data["allExData"] = allExchangeSlice

	if len(currencyPairs) == 0 {
		return nil, fmt.Errorf("No currency pairs found. Try running dcrextdata then try again.")
	}
	data["currencyPairs"] = currencyPairs

	if selectedExchange == "" && viewOption == "table" {
		selectedExchange = "All"
	} else if selectedExchange == "" && viewOption == "chart" {
		if len(allExchangeSlice) > 0 {
			selectedExchange = allExchangeSlice[0].Name
		} else {
			return nil, fmt.Errorf("No exchange source data. Try running dcrextdata then try again.")
		}
	}
	data["selectedExchange"] = selectedExchange

	if viewOption == "chart" {
		return data, nil
	}

	allExchangeTicksSlice, totalCount, err := s.db.FetchExchangeTicks(ctx, selectedCurrencyPair, selectedExchange, selectedInterval, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("Error in fetching exchange ticks, %s", err.Error())
	}

	if len(allExchangeTicksSlice) == 0 {
		data["message"] = fmt.Sprintf("%s %s", strings.Title(selectedExchange), noDataMessage)
		return data, nil
	}

	data["exData"] = allExchangeTicksSlice
	data["totalPages"] = int(math.Ceil(float64(totalCount) / float64(pageSize)))

	totalTxLoaded := offset + len(allExchangeTicksSlice)
	if int64(totalTxLoaded) < totalCount {
		data["nextPage"] = pageToLoad + 1
	}

	return data, nil
}

func (s *TickHub) getExchangeChartData(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	selectedTick := r.FormValue("selected-tick")
	selectedCurrencyPair := r.FormValue("selected-currency-pair")
	selectedInterval := r.FormValue("selected-interval")
	selectedExchange := r.FormValue("selected-exchange")

	data := map[string]interface{}{}

	ctx := r.Context()
	interval, err := strconv.Atoi(selectedInterval)
	if err != nil {
		web.RenderErrorfJSON(w, "Invalid interval, %s", err.Error())
		return
	}

	chartData, err := s.store.ExchangeTicksChartData(ctx, selectedTick, selectedCurrencyPair, interval, selectedExchange)
	if err != nil {
		web.RenderErrorfJSON(w, "Cannot fetch chart data, %s", err.Error())
		return
	}
	if len(chartData) == 0 {
		web.RenderErrorfJSON(w, "No data to generate %s chart.", selectedExchange)
		return
	}

	data["chartData"] = chartData

	web.RenderJSON(w, data)
}

func (s *TickHub) tickIntervalsByExchangeAndPair(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	selectedCurrencyPair := r.FormValue("currency-pair")
	var result = []struct {
		Label string `json:"label"`
		Value int    `json:"value"`
	}{
		{Label: "All", Value: -1},
	}
	pairs, err := s.store.TickIntervalsByExchangeAndPair(r.Context(), r.FormValue("exchange"), selectedCurrencyPair)
	if err != nil {
		if err.Error() != sql.ErrNoRows.Error() {
			web.RenderJSON(w, "error in loading intervals, "+err.Error())
			return
		}
		web.RenderJSON(w, result)
		return
	}

	for _, p := range pairs {
		result = append(result, struct {
			Label string `json:"label"`
			Value int    `json:"value"`
		}{
			Label: exchangeTickIntervals[p.Interval],
			Value: p.Interval,
		})
	}
	web.RenderJSON(w, result)
}

func (s *TickHub) currencyPairByExchange(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var result = []string{"All"}
	pairs, err := s.store.CurrencyPairByExchange(r.Context(), r.FormValue("exchange"))
	if err != nil {
		if err.Error() != sql.ErrNoRows.Error() {
			web.RenderErrorfJSON(w, "error in loading intervals, "+err.Error())
			return
		}
		web.RenderJSON(w, result)
		return
	}
	for _, p := range pairs {
		result = append(result, p.CurrencyPair)
	}
	web.RenderJSON(w, result)
}
