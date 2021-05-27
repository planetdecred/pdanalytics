package stats

import (
	"context"
	"io"
	"net/http"

	"github.com/planetdecred/pdanalytics/web"
)

type stat struct {
	server *web.Server
	db     store
}

type store interface {
	MempoolCount(ctx context.Context) (int64, error)
	BlockCount(ctx context.Context) (int64, error)
	VotesCount(ctx context.Context) (int64, error)
	PowCount(ctx context.Context) (int64, error)
	VspTickCount(ctx context.Context) (int64, error)
	ExchangeTickCount(ctx context.Context) (int64, error)
}

func Activate(server *web.Server, db store) error {
	st := stat{
		server: server,
		db:     db,
	}

	if err := st.server.Templates.AddTemplate("stats"); err != nil {
		return err
	}

	st.server.AddRoute("/stats", web.GET, st.statsPage)

	st.server.AddMenuItem(web.MenuItem{
		Href:      "/stats",
		HyperText: "Stats",
		Info:      "DB stats about pdanalytics",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "DB stats about pdanalytics",
		},
	})

	return nil
}

func (s stat) statsPage(w http.ResponseWriter, r *http.Request) {
	mempoolCount, err := s.db.MempoolCount(r.Context())
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		s.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	blocksCount, err := s.db.BlockCount(r.Context())
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		s.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)

		return
	}

	votesCount, err := s.db.VotesCount(r.Context())
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		s.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)

		return
	}

	powCount, err := s.db.PowCount(r.Context())
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		s.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	vspCount, err := s.db.VspTickCount(r.Context())
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		s.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	exchangeCount, err := s.db.ExchangeTickCount(r.Context())
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		s.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	data := map[string]interface{}{
		"mempoolCount": mempoolCount,
		"blocksCount":  blocksCount,
		"votesCount":   votesCount,
		"powCount":     powCount,
		"vspCount":     vspCount,
		"exchangeTick": exchangeCount,
	}

	str, err := s.server.Templates.ExecTemplateToString("stats", struct {
		*web.CommonPageData
		Data            map[string]interface{}
		BreadcrumbItems []web.BreadcrumbItem
	}{
		CommonPageData: s.server.CommonData(r),
		Data:           data,
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "DB stats about pdanalytics",
				Active:    true,
			},
		},
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		s.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}
