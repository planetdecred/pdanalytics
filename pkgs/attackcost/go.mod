module github.com/planetdecred/pdanalytics/pkgs/attackcost

go 1.13

require (
	github.com/decred/dcrd/chaincfg/v2 v2.3.0
	github.com/decred/dcrd/rpcclient/v5 v5.0.1
	github.com/decred/dcrd/wire v1.4.0
	github.com/decred/dcrdata/db/dbtypes v1.1.0
	github.com/decred/dcrdata/db/dbtypes/v2 v2.2.1
	github.com/decred/dcrdata/exchanges/v2 v2.1.0
	github.com/decred/dcrdata/explorer/types/v2 v2.1.1
	github.com/decred/slog v1.1.0
	github.com/planetdecred/pdanalytics/web v0.0.0-20210121232737-d068a16f7d67
)

replace (
	github.com/planetdecred/pdanalytics/web => ../../web
)
