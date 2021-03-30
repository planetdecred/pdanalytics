package agendas

import (
	"io"
	"net/http"

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
		Agendas       []*AgendaTagged
		VotingSummary *VoteSummary
	}{
		CommonPageData: exp.server.CommonData(r),
		Agendas:        agenda,
		VotingSummary:  exp.voteTracker.Summary(),
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
