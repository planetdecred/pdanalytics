package treasury

type Treasury struct {
	server *web.Server
}

func Activate(Context ctx, webServer *web.Server) error {

	treasury := &Treasury{
		server: webServer,
	}

	treasury.server.AddRoute("/proposals")
}
