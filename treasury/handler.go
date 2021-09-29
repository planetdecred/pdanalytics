package treasury

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/planetdecred/pdanalytics/web"
)






// TreasuryPage is the page handler for the "/treasury" path
func (exp *explorerUI) TreasuryPage(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), ctxAddress, exp.pageData.HomeInfo.DevAddress)
	r = r.WithContext(ctx)
	if queryVals := r.URL.Query(); queryVals.Get("txntype") == "" {
		// TODO: Change default to "tspend" once there are some tspends.
		queryVals.Set("txntype", "all")
		r.URL.RawQuery = queryVals.Encode()
	}

	limitN := defaultAddressRows
	if nParam := r.URL.Query().Get("n"); nParam != "" {
		val, err := strconv.ParseUint(nParam, 10, 64)
		if err != nil {
			exp.StatusPage(w, defaultErrorCode, "invalid n value", "", ExpStatusError)
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
			exp.StatusPage(w, defaultErrorCode, "invalid start value", "", ExpStatusError)
			return
		}
		offset = int64(val)
	}

	// Transaction types to show.
	txTypeStr := r.URL.Query().Get("txntype")
	txType := parseTreasuryTransactionType(txTypeStr)

	txns, err := exp.dataSource.TreasuryTxns(limitN, offset, txType)
	if exp.timeoutErrorPage(w, err, "TreasuryTxns") {
		return
	} else if err != nil {
		exp.StatusPage(w, defaultErrorCode, err.Error(), "", ExpStatusError)
		return
	}

	exp.pageData.RLock()
	treasuryBalance := exp.pageData.HomeInfo.TreasuryBalance
	exp.pageData.RUnlock()

	typeCount := treasuryTypeCount(treasuryBalance, txType)

	treasuryData := &TreasuryInfo{
		Net:             exp.ChainParams.Net.String(),
		MaxTxLimit:      MaxTreasuryRows,
		Path:            r.URL.Path,
		Limit:           limitN,
		Offset:          offset,
		TxnType:         txTypeStr,
		NumTransactions: int64(len(txns)),
		Transactions:    txns,
		Balance:         treasuryBalance,
		TypeCount:       typeCount,
	}

	xcBot := exp.xcBot
	if xcBot != nil {
		treasuryData.ConvertedBalance = xcBot.Conversion(math.Round(float64(treasuryBalance.Balance) / 1e8))
	}

	// Execute the HTML template.
	linkTemplate := fmt.Sprintf("/treasury?start=%%d&n=%d&txntype=%v", limitN, txType)
	pageData := struct {
		*CommonPageData
		Data        *TreasuryInfo
		FiatBalance *exchanges.Conversion
		Pages       []pageNumber
	}{
		CommonPageData: exp.commonData(r),
		Data:           treasuryData,
		FiatBalance:    exp.xcBot.Conversion(dcrutil.Amount(treasuryBalance.Balance).ToCoin()),
		Pages:          calcPages(int(typeCount), int(limitN), int(offset), linkTemplate),
	}
	str, err := exp.templates.exec("treasury", pageData)
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.StatusPage(w, defaultErrorCode, defaultErrorMessage, "", ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Turbolinks-Location", r.URL.RequestURI())
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// AddressPage is the page handler for the "/address" path.
func (exp *explorerUI) AddressPage(w http.ResponseWriter, r *http.Request) {
	// AddressPageData is the data structure passed to the HTML template
	type AddressPageData struct {
		*CommonPageData
		Data         *dbtypes.AddressInfo
		CRLFDownload bool
		FiatBalance  *exchanges.Conversion
		Pages        []pageNumber
	}

	// Grab the URL query parameters
	address, txnType, limitN, offsetAddrOuts, err := parseAddressParams(r)
	if err != nil {
		exp.StatusPage(w, defaultErrorCode, err.Error(), address, ExpStatusError)
		return
	}

	// Validate the address.
	addr, addrType, addrErr := txhelpers.AddressValidation(address, exp.ChainParams)
	isZeroAddress := addrErr == txhelpers.AddressErrorZeroAddress
	if addrErr != nil && !isZeroAddress {
		var status expStatus
		var message string
		code := defaultErrorCode
		switch addrErr {
		case txhelpers.AddressErrorDecodeFailed, txhelpers.AddressErrorUnknown:
			status = ExpStatusBadRequest
			message = "Unexpected issue validating this address."
		case txhelpers.AddressErrorWrongNet:
			status = ExpStatusWrongNetwork
			message = fmt.Sprintf("The address %v is valid on %s, not %s.",
				addr, exp.ChainParams.Net.String(), exp.NetName)
			code = wrongNetwork
		default:
			status = ExpStatusError
			message = "Unknown error."
		}

		exp.StatusPage(w, code, message, address, status)
		return
	}

	// Handle valid but unsupported address types.
	switch addrType {
	case txhelpers.AddressTypeP2PKH, txhelpers.AddressTypeP2SH:
		// All good.
	case txhelpers.AddressTypeP2PK:
		message := "Looks like you are searching for an address of type P2PK."
		exp.StatusPage(w, defaultErrorCode, message, address, ExpStatusP2PKAddress)
		return
	default:
		message := "Unsupported address type."
		exp.StatusPage(w, defaultErrorCode, message, address, ExpStatusNotSupported)
		return
	}

	// Retrieve address information from the DB and/or RPC.
	var addrData *dbtypes.AddressInfo
	if isZeroAddress {
		// For the zero address (e.g. DsQxuVRvS4eaJ42dhQEsCXauMWjvopWgrVg),
		// short-circuit any queries.
		addrData = &dbtypes.AddressInfo{
			Address:         address,
			Net:             exp.ChainParams.Net.String(),
			IsDummyAddress:  true,
			Balance:         new(dbtypes.AddressBalance),
			UnconfirmedTxns: new(dbtypes.AddressTransactions),
		}
	} else {
		addrData, err = exp.AddressListData(address, txnType, limitN, offsetAddrOuts)
		if exp.timeoutErrorPage(w, err, "TicketsPriceByHeight") {
			return
		} else if err != nil {
			exp.StatusPage(w, defaultErrorCode, err.Error(), address, ExpStatusError)
			return
		}
	}

	// Set page parameters.
	addrData.IsDummyAddress = isZeroAddress // may be redundant
	addrData.Path = r.URL.Path

	// If exchange monitoring is active, prepare a fiat balance conversion
	conversion := exp.xcBot.Conversion(dcrutil.Amount(addrData.Balance.TotalUnspent).ToCoin())

	// For Windows clients only, link to downloads with CRLF (\r\n) line
	// endings.
	UseCRLF := strings.Contains(r.UserAgent(), "Windows")

	if limitN == 0 {
		limitN = 20
	}

	linkTemplate := fmt.Sprintf("/address/%s?start=%%d&n=%d&txntype=%v", addrData.Address, limitN, txnType)

	// Execute the HTML template.
	pageData := AddressPageData{
		CommonPageData: exp.commonData(r),
		Data:           addrData,
		CRLFDownload:   UseCRLF,
		FiatBalance:    conversion,
		Pages:          calcPages(int(addrData.TxnCount), int(limitN), int(offsetAddrOuts), linkTemplate),
	}
	str, err := exp.templates.exec("address", pageData)
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.StatusPage(w, defaultErrorCode, defaultErrorMessage, "", ExpStatusError)
		return
	}

	log.Tracef(`"address" template HTML size: %.2f kiB (%s, %v, %d)`,
		float64(len(str))/1024.0, address, txnType, addrData.NumTransactions)

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Turbolinks-Location", r.URL.RequestURI())
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// TreasuryTable is the handler for the "/treasurytable" path.
func (exp *explorerUI) TreasuryTable(w http.ResponseWriter, r *http.Request) {
	// Grab the URL query parameters
	txType, limitN, offset, err := parseTreasuryParams(r)
	if err != nil {
		log.Errorf("TreasuryTable request error: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	txns, err := exp.dataSource.TreasuryTxns(limitN, offset, txType)
	if exp.timeoutErrorPage(w, err, "TreasuryTxns") {
		return
	} else if err != nil {
		exp.StatusPage(w, defaultErrorCode, err.Error(), "", ExpStatusError)
		return
	}

	exp.pageData.RLock()
	bal := exp.pageData.HomeInfo.TreasuryBalance
	exp.pageData.RUnlock()

	linkTemplate := "/treasury" + "?start=%d&n=" + strconv.FormatInt(limitN, 10) + "&txntype=" + fmt.Sprintf("%v", txType)

	response := struct {
		TxnCount int64        `json:"tx_count"`
		HTML     string       `json:"html"`
		Pages    []pageNumber `json:"pages"`
	}{
		TxnCount: bal.TxCount, // + addrData.ImmatureCount,
		Pages:    calcPages(int(treasuryTypeCount(bal, txType)), int(limitN), int(offset), linkTemplate),
	}

	type txData struct {
		Transactions []*dbtypes.TreasuryTx
	}

	response.HTML, err = exp.templates.exec("treasurytable", struct {
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
