package homepage

import (
	"io"
	"net/http"

	"github.com/planetdecred/pdanalytics/attackcost"
	"github.com/planetdecred/pdanalytics/charts"
	"github.com/planetdecred/pdanalytics/parameters"
	"github.com/planetdecred/pdanalytics/stakingreward"
	"github.com/planetdecred/pdanalytics/web"
)

type Home struct {
	server *web.Server
	mods   Mods
}

type Mods struct {
	Ac    *attackcost.Attackcost
	Stk   *stakingreward.Calculator
	Prm   *parameters.Parameters
	Chrts *charts.Charts
}

func New(server *web.Server, mods Mods) (*Home, error) {
	hm := &Home{
		server: server,
		mods:   mods,
	}
	err := server.Templates.AddTemplate("home")

	if err != nil {
		return nil, err
	}

	server.AddRoute("/", web.GET, hm.homepage)
	return hm, nil
}

func (hm *Home) homepage(w http.ResponseWriter, r *http.Request) {
	stk := hm.mods.Stk != nil
	ac := hm.mods.Ac != nil
	prm := hm.mods.Prm != nil
	chrts := hm.mods.Chrts != nil
	str, err := hm.server.Templates.ExecTemplateToString("home", struct {
		*web.CommonPageData
		NoModEnabled         bool
		StakingRewardEnabled bool
		ParametersEnabled    bool
		AttackCostEnabled    bool
		ChartsEnabled        bool
		BreadcrumbItems      []web.BreadcrumbItem
	}{
		NoModEnabled:         !(stk || prm || ac || chrts),
		CommonPageData:       hm.server.CommonData(r),
		StakingRewardEnabled: stk,
		ParametersEnabled:    prm,
		AttackCostEnabled:    ac,
		ChartsEnabled:        chrts,
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Home",
				Active:    true,
			},
		},
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		hm.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, "", web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}
