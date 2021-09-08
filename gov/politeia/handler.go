package politeia

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	pitypes "github.com/decred/dcrdata/gov/v4/politeia/types"
	ticketvotev1 "github.com/decred/politeia/politeiawww/api/ticketvote/v1"
	"github.com/go-chi/chi"
	"github.com/planetdecred/pdanalytics/dbhelper"
	"github.com/planetdecred/pdanalytics/web"
)

// ProposalsPage is the page handler for the "/proposals" path.
func (prop *proposals) ProposalsPage(w http.ResponseWriter, r *http.Request) {
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
	var proposals []*pitypes.ProposalRecord

	// Check if filter by votes status query parameter was passed.
	if filterBy > 0 {
		proposals, count, err = prop.db.ProposalsAll(int(offset),
			int(rowsCount), int(filterBy))
	} else {
		proposals, count, err = prop.db.ProposalsAll(int(offset),
			int(rowsCount))
	}

	if err != nil {
		log.Errorf("Cannot fetch proposals: %v", err)
		prop.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	lastOffsetRows := uint64(count) % rowsCount
	var lastOffset uint64

	if lastOffsetRows == 0 && uint64(count) > rowsCount {
		lastOffset = uint64(count) - rowsCount
	} else if lastOffsetRows > 0 && uint64(count) > rowsCount {
		lastOffset = uint64(count) - lastOffsetRows
	}

	// Parse vote statuses map with only used status by the UI. Also
	// capitalizes first letter of the string status format.
	votesStatus := map[ticketvotev1.VoteStatusT]string{
		ticketvotev1.VoteStatusUnauthorized: "Unauthorized",
		ticketvotev1.VoteStatusAuthorized:   "Authorized",
		ticketvotev1.VoteStatusStarted:      "Started",
		ticketvotev1.VoteStatusFinished:     "Finished",
		ticketvotev1.VoteStatusApproved:     "Approved",
		ticketvotev1.VoteStatusRejected:     "Rejected",
		ticketvotev1.VoteStatusIneligible:   "Ineligible",
	}

	str, err := prop.server.Templates.ExecTemplateToString("proposals", struct {
		*web.CommonPageData
		Proposals       []*pitypes.ProposalRecord
		VotesStatus     map[ticketvotev1.VoteStatusT]string
		VStatusFilter   int
		Offset          int64
		Limit           int64
		TotalCount      int64
		LastOffset      int64
		PoliteiaURL     string
		LastPropSync    int64
		TimePerBlock    int64
		Height          uint32
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: prop.server.CommonData(r),
		Proposals:      proposals,
		VotesStatus:    votesStatus,
		Offset:         int64(offset),
		Limit:          int64(rowsCount),
		VStatusFilter:  int(filterBy),
		TotalCount:     int64(count),
		LastOffset:     int64(lastOffset),
		PoliteiaURL:    prop.politeiaURL,
		LastPropSync:   prop.db.ProposalsLastSync(),
		TimePerBlock:   int64(prop.client.Params.TargetTimePerBlock.Seconds()),
		Height:         prop.height,
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Proposals",
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
	io.WriteString(w, str)
}

// ProposalPage is the page handler for the "/proposal" path.
func (prop *proposals) ProposalPage(w http.ResponseWriter, r *http.Request) {
	// Attempts to retrieve a proposal tonken from the URL path.
	param := getProposalTokenCtx(r)
	proposalInfo, err := prop.db.ProposalByToken(param)
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		prop.server.StatusPage(w, r, web.DefaultErrorCode, "the proposal token or RefID does not exist", "", web.ExpStatusNotFound)
		return
	}

	commonData := prop.server.CommonData(r)
	str, err := prop.server.Templates.ExecTemplateToString("proposal", struct {
		*web.CommonPageData
		Data            *pitypes.ProposalRecord
		PoliteiaURL     string
		ShortToken      string
		Metadata        *pitypes.ProposalMetadata
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: commonData,
		Data:           proposalInfo,
		PoliteiaURL:    prop.politeiaURL,
		ShortToken:     proposalInfo.Token[0:7],
		Metadata: proposalInfo.Metadata(int64(commonData.Tip.Height),
			int64(prop.client.Params.TargetTimePerBlock/time.Second)),
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Proposals",
				Href:      "/proposals",
			},
			{
				HyperText: proposalInfo.Name,
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
	io.WriteString(w, str)
}

func (prop *proposals) getProposalChartData(w http.ResponseWriter, r *http.Request) {
	token := getProposalTokenCtx(r)

	proposal, err := prop.db.ProposalByToken(token)
	if dbhelper.IsTimeoutErr(err) {
		log.Errorf("ProposalVotes: %v", err)
		http.Error(w, "Database timeout.", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		log.Errorf("Unable to get proposals chart data for token %s : %v", token, err)
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity),
			http.StatusUnprocessableEntity)
		return
	}

	web.RenderJSON(w, proposal.ChartData)
}

func getProposalPathCtx(r *http.Request) string {
	hash, ok := r.Context().Value(web.CtxProposalToken).(string)
	if !ok {
		log.Trace("Proposal token not set")
		return ""
	}
	return hash
}

// proposalPathCtx embeds "proposaltoken" into the request context
func proposalPathCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proposalToken := chi.URLParam(r, "proposaltoke")
		ctx := context.WithValue(r.Context(), web.CtxProposalToken, proposalToken)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// proposalTokenCtx returns a http.HandlerFunc that embeds the value at the url
// part {token} into the request context
func proposalTokenCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := chi.URLParam(r, "token")
		ctx := context.WithValue(r.Context(), web.CtxProposalToken, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getProposalTokenCtx retrieves the ctxProposalToken data from the request context.
// If the value is not set, an empty string is returned.
func getProposalTokenCtx(r *http.Request) string {
	tp, ok := r.Context().Value(web.CtxProposalToken).(string)
	if !ok {
		log.Trace("proposal token hash not set")
		return ""
	}
	return tp
}
