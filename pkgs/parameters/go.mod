module github.com/decred/dcrdata/pkgs/parameters

go 1.13

require (
	github.com/decred/dcrd/chaincfg v1.5.2
	github.com/decred/dcrd/chaincfg/v2 v2.3.0
	github.com/decred/dcrd/rpcclient v1.1.0
	github.com/decred/dcrd/rpcclient/v5 v5.0.0
	github.com/decred/dcrdata/db/dbtypes/v2 v2.2.1
	github.com/decred/dcrdata/explorer/types v1.1.0
	github.com/decred/dcrdata/explorer/types/v2 v2.1.1
	github.com/decred/slog v1.1.0
	github.com/planetdecred/pdanalytics/web v0.0.0-00010101000000-000000000000
	google.golang.org/appengine v1.6.7
)

replace github.com/planetdecred/pdanalytics/web => ../../web
