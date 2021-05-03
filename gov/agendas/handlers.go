package agendas

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/planetdecred/pdanalytics/web"
)

// AgendasPage is the page handler for the "/agendas" path.
func (exp *agendas) AgendasPage(w http.ResponseWriter, r *http.Request) {
	if exp.voteTracker == nil {
		log.Warnf("Agendas requested with nil voteTracker")
		exp.server.StatusPage(w, r, "", "agendas disabled on simnet", "", web.ExpStatusPageDisabled)
		return
	}

	agenda, err := exp.agendasSource.AllAgendas()
	if err != nil {
		log.Errorf("Error fetching agendas: %v", err)
		exp.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	str, err := exp.server.Templates.ExecTemplateToString("agendas", struct {
		*web.CommonPageData
		Agendas         []*AgendaTagged
		VotingSummary   *VoteSummary
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: exp.server.CommonData(r),
		Agendas:        agenda,
		VotingSummary:  exp.voteTracker.Summary(),
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Agendas",
				Active:    true,
			},
		},
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

// AgendaPage is the page handler for the "/agenda" path.
func (exp *agendas) AgendaPage(w http.ResponseWriter, r *http.Request) {
	errPageInvalidAgenda := func(err error) {
		log.Errorf("Template execute failure: %v", err)
		exp.server.StatusPage(w, r, web.DefaultErrorCode, "the agenda ID given seems to not exist",
			"", web.ExpStatusNotFound)
	}

	// Attempt to get agendaid string from URL path.
	agendaId := getAgendaIDCtx(r)
	agendaInfo, err := exp.agendasSource.AgendaInfo(agendaId)
	if err != nil {
		errPageInvalidAgenda(err)
		return
	}

	summary, err := exp.dataSource.AgendasVotesSummary(agendaId)
	if err != nil {
		log.Errorf("fetching Cumulative votes choices count failed: %v", err)
	}

	// Overrides the default count value with the actual vote choices count
	// matching data displayed on "Cumulative Vote Choices" and "Vote Choices By
	// Block" charts.
	var totalVotes uint32
	for index := range agendaInfo.Choices {
		switch strings.ToLower(agendaInfo.Choices[index].ID) {
		case "abstain":
			agendaInfo.Choices[index].Count = summary.Abstain
		case "yes":
			agendaInfo.Choices[index].Count = summary.Yes
		case "no":
			agendaInfo.Choices[index].Count = summary.No
		}
		totalVotes += agendaInfo.Choices[index].Count
	}

	ruleChangeI := exp.client.Params.RuleChangeActivationInterval
	qVotes := uint32(float64(ruleChangeI) * agendaInfo.QuorumProgress)

	var timeLeft string
	blocksLeft := summary.LockedIn - int64(exp.height)

	if blocksLeft > 0 {
		// Approximately 1 block per 5 minutes.
		var minPerblock = 5 * time.Minute

		hoursLeft := int((time.Duration(blocksLeft) * minPerblock).Hours())
		if hoursLeft > 0 {
			timeLeft = fmt.Sprintf("%v days %v hours", hoursLeft/24, hoursLeft%24)
		}
	} else {
		blocksLeft = 0
	}

	str, err := exp.server.Templates.ExecTemplateToString("agenda", struct {
		*web.CommonPageData
		Ai              *AgendaTagged
		QuorumVotes     uint32
		RuleChangeI     uint32
		VotingStarted   int64
		LockedIn        int64
		BlocksLeft      int64
		TimeRemaining   string
		TotalVotes      uint32
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: exp.server.CommonData(r),
		Ai:             agendaInfo,
		QuorumVotes:    qVotes,
		RuleChangeI:    ruleChangeI,
		VotingStarted:  summary.VotingStarted,
		LockedIn:       summary.LockedIn,
		BlocksLeft:     blocksLeft,
		TimeRemaining:  timeLeft,
		TotalVotes:     totalVotes,
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Agendas",
				Href:      "/agendas",
			},
			{
				HyperText: agendaInfo.ID,
				Active:    true,
			},
		},
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

// proposalPathCtx embeds "proposalrefID" into the request context
func agendaPathCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agendaID := chi.URLParam(r, "id")
		ctx := context.WithValue(r.Context(), web.CtxAgendaId, agendaID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getAgendaIDCtx(r *http.Request) string {
	hash, ok := r.Context().Value(web.CtxAgendaId).(string)
	if !ok {
		log.Trace("Agendaid not set")
		return ""
	}
	return hash
}
