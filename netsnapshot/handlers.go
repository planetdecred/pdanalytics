package netsnapshot

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/planetdecred/pdanalytics/web"
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
		data map[string]interface{}
	}{
		CommonPageData: t.server.CommonData(r),
		data:           data,
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
