package treasury

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/planetdecred/pdanalytics/web"

	"github.com/decred/dcrd/blockchain/stake/v4"
	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/decred/dcrdata/v7/db/dbtypes"
)

const (
	defaultAddressRows int64  = 20
	MaxTreasuryRows    int64  = 200
	MaxAddressRows     int64  = 1000
	txURL              string = "treasury/tx"
	balURL             string = "treasury/balance"
	ellipsisHTML       string = "…"
)

// TreasuryPage is the page handler for the "/treasury" path
func (trs *Treasury) TreasuryPage(w http.ResponseWriter, r *http.Request) {
	if queryVals := r.URL.Query(); queryVals.Get("txntype") == "" {
		queryVals.Set("txntype", "all")
		r.URL.RawQuery = queryVals.Encode()
	}

	limitN := defaultAddressRows
	if nParam := r.URL.Query().Get("n"); nParam != "" {
		val, err := strconv.ParseUint(nParam, 10, 64)
		if err != nil {
			trs.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
			return
		}
		if int64(val) > MaxTreasuryRows {
			log.Warnf("TreasuryPage: requested up to %d address rows, "+
				"limiting to %d", limitN, MaxTreasuryRows)
			limitN = MaxTreasuryRows
		} else {
			limitN = int64(val)
		}
	}
	// Number of txns to skip (OFFSET in database query). For UX reasons, the
	// "start" URL query parameter is used.
	var offset int64
	if startParam := r.URL.Query().Get("start"); startParam != "" {
		val, err := strconv.ParseUint(startParam, 10, 64)
		if err != nil {
			trs.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
			return
		}
		offset = int64(val)
	}

	// Transaction types to show.
	txTypeStr := r.URL.Query().Get("txntype")
	txType := parseTreasuryTransactionType(txTypeStr)

	txTemp, err := trs.TreasuryTxns(TxParams{
		Limit:  limitN,
		Offset: offset,
		TxType: txType,
	})

	txns := txTemp.Txns
	if err != nil {
		trs.server.StatusPage(w, r, web.DefaultErrorCode, err.Error(), "", web.ExpStatusError)
		return
	}

	bal, _ := trs.TreasuryBalance()
	treasuryBalance := bal.Balance

	typeCount := treasuryTypeCount(treasuryBalance, txType)

	treasuryData := &TreasuryInfo{
		Net:             trs.server.CommonData(r).ChainParams.Net.String(),
		MaxTxLimit:      MaxTreasuryRows,
		Path:            r.URL.Path,
		Limit:           limitN,
		Offset:          offset,
		TxnType:         txTypeStr,
		NumTransactions: int64(len(txns)),
		Transactions:    txns,
		Balance:         treasuryBalance,
		TypeCount:       typeCount,
		APIURL:          trs.aPIURL,
	}

	xcBot := trs.xcBot
	if xcBot != nil {
		treasuryData.ConvertedBalance = xcBot.Conversion(math.Round(float64(treasuryBalance.Balance) / 1e8))
	}

	// Execute the HTML template.
	linkTemplate := fmt.Sprintf("/treasury?start=%%d&n=%d&txntype=%v", limitN, txType)
	pageData := struct {
		*web.CommonPageData
		Data            *TreasuryInfo
		FiatBalance     *exchanges.Conversion
		Pages           []PageNumber
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: trs.server.CommonData(r),
		Data:           treasuryData,
		FiatBalance:    trs.xcBot.Conversion(dcrutil.Amount(treasuryBalance.Balance).ToCoin()),
		Pages:          calcPages(int(typeCount), int(limitN), int(offset), linkTemplate),
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Treasury",
				Active:    true,
			},
		},
	}
	str, err := trs.server.Templates.Exec("treasury", pageData)
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		trs.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Turbolinks-Location", r.URL.RequestURI())
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// TreasuryTable is the handler for the "/treasurytable" path.
func (trs *Treasury) TreasuryTable(w http.ResponseWriter, r *http.Request) {
	// Grab the URL query parameters
	txType, limitN, offset, err := parseTreasuryParams(r)
	if err != nil {
		log.Errorf("TreasuryTable request error: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	tx, err := trs.TreasuryTxns(TxParams{
		Limit:  limitN,
		Offset: offset,
		TxType: txType,
	})

	txns := tx.Txns
	if err != nil {
		trs.server.StatusPage(w, r, web.DefaultErrorCode, err.Error(), "", web.ExpStatusError)
		return
	}

	b, _ := trs.TreasuryBalance()
	bal := b.Balance

	linkTemplate := "/treasury" + "?start=%d&n=" + strconv.FormatInt(limitN, 10) + "&txntype=" + fmt.Sprintf("%v", txType)

	response := struct {
		TxnCount int64        `json:"tx_count"`
		HTML     string       `json:"html"`
		Pages    []PageNumber `json:"pages"`
	}{
		TxnCount: bal.TxCount, // + addrData.ImmatureCount,
		Pages:    calcPages(int(treasuryTypeCount(bal, txType)), int(limitN), int(offset), linkTemplate),
	}

	type txData struct {
		Transactions []*TreasuryTx
	}

	response.HTML, err = trs.server.Templates.Exec("treasurytable", struct {
		Data txData
	}{
		Data: txData{
			Transactions: txns,
		},
	})
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	log.Tracef(`"treasurytable" template HTML size: %.2f kiB (%v, %d)`,
		float64(len(response.HTML))/1024.0, txType, len(txns))

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	//enc.SetEscapeHTML(false)
	err = enc.Encode(response)
	if err != nil {
		log.Debug(err)
	}
}

func (trs *Treasury) TreasuryTxns(params TxParams) (Txns, error) {
	client := trs.client
	txns := Txns{}
	paramJson, err := json.Marshal(params)
	if err != nil {
		return txns, err
	}

	req, err := http.NewRequest("POST", trs.aPIURL+txURL, bytes.NewBuffer(paramJson))
	if err != nil {
		return txns, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return txns, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return txns, err
	}

	err = json.Unmarshal(body, &txns)
	if err != nil {
		return txns, err
	}

	return txns, nil
}

func (trs *Treasury) TreasuryBalance() (Balance, error) {
	client := trs.client
	bal := Balance{}

	req, err := http.NewRequest("GET", trs.aPIURL+balURL, nil)
	if err != nil {
		return bal, err
	}

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return bal, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return bal, err
	}

	err = json.Unmarshal(body, &bal)
	if err != nil {
		return bal, err
	}

	return bal, nil
}

// parseTreasuryTransactionType parses a treasury transaction type from a
// string. If the provided string is not recognized as a treasury type, the
// special value -1, representing "all", will be returned.
func parseTreasuryTransactionType(txnTypeStr string) (txType stake.TxType) {
	switch strings.ToLower(txnTypeStr) {
	case "tspend":
		return stake.TxTypeTSpend
	case "tadd":
		return stake.TxTypeTAdd
	case "treasurybase":
		return stake.TxTypeTreasuryBase
	}
	return stake.TxType(-1)
}

// treasuryTypeCount returns the tx count for the type treasury tx type. The
// special value txType = -1 specifies all types combined.
func treasuryTypeCount(treasuryBalance *dbtypes.TreasuryBalance, txType stake.TxType) int64 {
	typedCount := treasuryBalance.TxCount
	switch txType {
	case stake.TxTypeTSpend:
		typedCount = treasuryBalance.SpendCount
	case stake.TxTypeTAdd:
		typedCount = treasuryBalance.AddCount
	case stake.TxTypeTreasuryBase:
		typedCount = treasuryBalance.TGenCount
	}
	return typedCount
}

// Get a set of pagination numbers, based on a set number of rows that are
// assumed to start from page 1 at the lowest row and ascend from there.
// For example, if there are 20 pages of 10 rows, 0 - 199, page 1 would start at
// row 0 and go up to row 9. If the offset is between 0 and 9, the
// pagination would return the pageNumbers  necessary to create a pagination
// That looks like 1 2 3 4 5 6 7 8 ... 20. The pageNumber includes a link with
// the offset inserted using Sprintf.
func calcPages(rows, pageSize, offset int, link string) PageNumbers {
	if pageSize == 0 {
		return PageNumbers{}
	}
	nums := make(PageNumbers, 0, 11)
	endIdx := rows / pageSize
	if endIdx == 0 {
		return nums
	}
	pages := endIdx + 1
	currentPageIdx := offset / pageSize

	if pages > 10 {
		nums = append(nums, makePageNumber(currentPageIdx == 0, fmt.Sprintf(link, 0), "1"))
		start := currentPageIdx - 3
		endMiddle := start + 6
		if start <= 1 {
			start = 1
			endMiddle = 7
		} else if endMiddle >= endIdx-1 {
			endMiddle = endIdx - 1
			start = endMiddle - 6
		}
		if start > 1 {
			nums = append(nums, makePageNumber(false, "", ellipsisHTML))
		}

		for i := start; i <= endMiddle; i++ {
			nums = append(nums, makePageNumber(i == currentPageIdx, fmt.Sprintf(link, i*pageSize), strconv.Itoa(i+1)))
		}
		if endMiddle < endIdx-1 {
			nums = append(nums, makePageNumber(false, "", ellipsisHTML))
		}
		if pages > 1 {
			nums = append(nums, makePageNumber(currentPageIdx == endIdx, fmt.Sprintf(link, endIdx*pageSize), strconv.Itoa(pages)))
		}
	} else {
		for i := 0; i < pages; i++ {
			nums = append(nums, makePageNumber(i == currentPageIdx, fmt.Sprintf(link, i*pageSize), strconv.Itoa(i+1)))
		}
	}

	return nums
}

func makePageNumber(active bool, link, str string) PageNumber {
	return PageNumber{
		Active: active,
		Link:   link,
		Str:    str,
	}
}

// parseTreasuryParams parses the tx filters for the treasury page. Used by both
// TreasuryPage and TreasuryTable.
func parseTreasuryParams(r *http.Request) (txType stake.TxType, limitN, offsetAddrOuts int64, err error) {
	tType, limitN, offsetAddrOuts, err := parsePaginationParams(r)
	txType = parseTreasuryTransactionType(tType)
	return
}

// parsePaginationParams parses the pagination parameters from the query. The
// txnType string is returned as-is. The caller must decipher the string.
func parsePaginationParams(r *http.Request) (txnType string, limitN, offset int64, err error) {
	// Number of outputs for the address to query the database for. The URL
	// query parameter "n" is used to specify the limit (e.g. "?n=20").
	limitN = defaultAddressRows

	if nParam := r.URL.Query().Get("n"); nParam != "" {

		var val uint64
		val, err = strconv.ParseUint(nParam, 10, 64)
		if err != nil {
			err = fmt.Errorf("invalid n value")
			return
		}
		if int64(val) > MaxAddressRows {
			log.Warnf("addressPage: requested up to %d address rows, "+
				"limiting to %d", limitN, MaxAddressRows)
			limitN = MaxAddressRows
		} else {
			limitN = int64(val)
		}
	}

	// Number of outputs to skip (OFFSET in database query). For UX reasons, the
	// "start" URL query parameter is used.
	if startParam := r.URL.Query().Get("start"); startParam != "" {
		var val uint64
		val, err = strconv.ParseUint(startParam, 10, 64)
		if err != nil {
			err = fmt.Errorf("invalid start value")
			return
		}
		offset = int64(val)
	}

	// Transaction types to show.
	txnType = r.URL.Query().Get("txntype")
	if txnType == "" {
		txnType = "all"
	}

	return
}
