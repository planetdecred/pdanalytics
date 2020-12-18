module github.com/planetdecred/pdanalytics/stakingreward

go 1.12

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/decred/dcrd/chaincfg v1.5.2 // indirect
	github.com/decred/dcrd/chaincfg/v2 v2.3.0
	github.com/decred/dcrd/dcrutil v1.3.0
	github.com/decred/dcrd/gcs v1.1.0 // indirect
	github.com/decred/dcrd/rpcclient v1.1.0
	github.com/decred/dcrd/rpcclient/v5 v5.0.0
	github.com/decred/dcrd/wire v1.3.0
	github.com/decred/dcrdata/db/dbtypes/v2 v2.2.1
	github.com/decred/dcrdata/exchanges v1.0.0
	github.com/decred/dcrdata/exchanges/v2 v2.1.0
	github.com/decred/dcrdata/explorer/types/v2 v2.1.1
	github.com/decred/dcrdata/txhelpers/v4 v4.0.1
	github.com/decred/slog v1.1.0
	github.com/planetdecred/pdanalytics/web v0.0.0-00010101000000-000000000000
	github.com/prometheus/common v0.15.0
)

replace github.com/planetdecred/pdanalytics/web => ../../web
