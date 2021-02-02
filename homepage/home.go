package homepage

import (
	"io"
	"net/http"

	"github.com/planetdecred/pdanalytics/attackcost"
	"github.com/planetdecred/pdanalytics/web"
)

type Home struct {
	server *web.Server
	mods   Mods
}

type Mods struct {
	ac *attackcost.Attackcost
}

func New(server *web.Server) (*Home, error) {
	hm := &Home{
		server: server,
	}
	err := server.Templates.AddTemplate("home")

	if err != nil {
		return nil, err
	}

	server.AddRoute("/", web.GET, hm.homepage)
	return hm, nil
}

func (hm *Home) homepage(w http.ResponseWriter, r *http.Request) {
	str, err := hm.server.Templates.ExecTemplateToString("home", struct {
		*web.CommonPageData
	}{
		CommonPageData: hm.server.CommonData(r),
	})

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}
