package politeia

import (
	"io"
	"net/http"
	"strconv"

	pitypes "github.com/planetdecred/pdanalytics/gov/politeia/types"
	"github.com/planetdecred/pdanalytics/web"
)

// ProposalsPage is the page handler for the "/proposals" path.
func (exp *proposals) ProposalsPage(w http.ResponseWriter, r *http.Request) {
	rowsCount := uint64(20)
	if rowsStr := r.URL.Query().Get("rows"); rowsStr != "" {
		val, err := strconv.ParseUint(rowsStr, 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if val > 0 {
			rowsCount = val
		}
	}
	var offset uint64
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		val, err := strconv.ParseUint(offsetStr, 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		offset = val
	}
	var filterBy uint64
	if filterByStr := r.URL.Query().Get("byvotestatus"); filterByStr != "" {
		val, err := strconv.ParseUint(filterByStr, 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		filterBy = val
	}

	var err error
	var count int
	var proposals []*pitypes.ProposalInfo

	// Check if filter by votes status query parameter was passed.
	if filterBy > 0 {
		proposals, count, err = exp.db.AllProposals(int(offset),
			int(rowsCount), int(filterBy))
	} else {
		proposals, count, err = exp.db.AllProposals(int(offset),
			int(rowsCount))
	}

	if err != nil {
		log.Errorf("Cannot fetch proposals: %v", err)
		exp.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	str, err := exp.server.Templates.ExecTemplateToString("proposals", struct {
		*web.CommonPageData
		Proposals     []*pitypes.ProposalInfo
		VotesStatus   map[pitypes.VoteStatusType]string
		VStatusFilter int
		Offset        int64
		Limit         int64
		TotalCount    int64
		PoliteiaURL   string
		LastVotesSync int64
		LastPropSync  int64
		TimePerBlock  int64
	}{
		CommonPageData: exp.server.CommonData(r),
		Proposals:      proposals,
		VotesStatus:    pitypes.VotesStatuses(),
		Offset:         int64(offset),
		Limit:          int64(rowsCount),
		VStatusFilter:  int(filterBy),
		TotalCount:     int64(count),
		PoliteiaURL:    exp.politeiaURL,
		LastVotesSync:  exp.dataSource.LastPiParserSync().UTC().Unix(),
		LastPropSync:   exp.db.LastProposalsSync(),
		TimePerBlock:   int64(exp.client.Params.TargetTimePerBlock.Seconds()),
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}
