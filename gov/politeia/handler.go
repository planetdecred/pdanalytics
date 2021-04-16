package politeia

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/planetdecred/pdanalytics/dbhelper"
	pitypes "github.com/planetdecred/pdanalytics/gov/politeia/types"
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
	if filterByStr := r.URL.Query().Get("byvotestatus"); filterByStr != "" && filterByStr != "all" {
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
		proposals, count, err = prop.db.AllProposals(int(offset),
			int(rowsCount), int(filterBy))
	} else {
		proposals, count, err = prop.db.AllProposals(int(offset),
			int(rowsCount))
	}

	if err != nil {
		log.Errorf("Cannot fetch proposals: %v", err)
		prop.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	str, err := prop.server.Templates.ExecTemplateToString("proposals", struct {
		*web.CommonPageData
		Proposals       []*pitypes.ProposalInfo
		VotesStatus     map[pitypes.VoteStatusType]string
		VStatusFilter   int
		Offset          int64
		Limit           int64
		TotalCount      int64
		PoliteiaURL     string
		LastVotesSync   int64
		LastPropSync    int64
		TimePerBlock    int64
		Height          uint32
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: prop.server.CommonData(r),
		Proposals:      proposals,
		VotesStatus:    pitypes.VotesStatuses(),
		Offset:         int64(offset),
		Limit:          int64(rowsCount),
		VStatusFilter:  int(filterBy),
		TotalCount:     int64(count),
		PoliteiaURL:    prop.politeiaURL,
		LastVotesSync:  prop.LastPiParserSync().UTC().Unix(),
		LastPropSync:   prop.db.LastProposalsSync(),
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
	// Attempts to retrieve a proposal refID from the URL path.
	param := getProposalPathCtx(r)
	proposalInfo, err := prop.db.ProposalByRefID(param)
	if err != nil {
		// Check if the URL parameter passed is a proposal token and attempt to
		// fetch its data.
		proposalInfo, newErr := prop.db.ProposalByToken(param)
		if newErr == nil && proposalInfo != nil && proposalInfo.RefID != "" {
			// redirect to a human readable url (replace the token with the RefID)
			http.Redirect(w, r, "/proposal/"+proposalInfo.RefID, http.StatusPermanentRedirect)
			return
		}

		log.Errorf("Template execute failure: %v", err)
		prop.server.StatusPage(w, r, web.DefaultErrorCode, "the proposal token or RefID does not exist", "", web.ExpStatusNotFound)
		return
	}

	commonData := prop.server.CommonData(r)
	str, err := prop.server.Templates.ExecTemplateToString("proposal", struct {
		*web.CommonPageData
		Data            *pitypes.ProposalInfo
		PoliteiaURL     string
		Metadata        *pitypes.ProposalMetadata
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: commonData,
		Data:           proposalInfo,
		PoliteiaURL:    prop.politeiaURL,
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
	votesData, err := prop.dataSource.ProposalVotes(r.Context(), token)
	if dbhelper.IsTimeoutErr(err) {
		log.Errorf("ProposalVotes: %v", err)
		http.Error(w, "Database timeout.", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		log.Errorf("Unable to get proposals votes for token %s : %v", token, err)
		http.Error(w, http.StatusText(http.StatusUnprocessableEntity),
			http.StatusUnprocessableEntity)
		return
	}

	web.RenderJSON(w, votesData)
}

func getProposalPathCtx(r *http.Request) string {
	hash, ok := r.Context().Value(web.CtxProposalRefID).(string)
	if !ok {
		log.Trace("Proposal ref ID not set")
		return ""
	}
	return hash
}

// proposalPathCtx embeds "proposalrefID" into the request context
func proposalPathCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proposalRefID := chi.URLParam(r, "proposalrefid")
		ctx := context.WithValue(r.Context(), web.CtxProposalRefID, proposalRefID)
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
