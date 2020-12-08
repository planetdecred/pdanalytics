module github.com/planetdecred/pdanalytics/attackcost

go 1.12

require (
	github.com/caarlos0/env v3.5.0+incompatible
	github.com/decred/dcrd/chaincfg/v2 v2.3.0
	github.com/decred/dcrd/dcrutil/v2 v2.0.1
	github.com/decred/dcrd/rpcclient v1.1.0
	github.com/decred/dcrd/rpcclient/v5 v5.0.1
	github.com/decred/dcrd/wire v1.4.0
	github.com/decred/dcrdata/blockdata/v5 v5.0.1
	github.com/decred/dcrdata/db/dbtypes/v2 v2.2.1
	github.com/decred/dcrdata/db/dcrpg/v5 v5.0.1
	github.com/decred/dcrdata/exchanges/v2 v2.1.0
	github.com/decred/dcrdata/explorer/types/v2 v2.1.1
	github.com/decred/dcrdata/gov/v3 v3.0.0
	github.com/decred/dcrdata/mempool/v5 v5.0.2
	github.com/decred/dcrdata/middleware/v3 v3.1.0
	github.com/decred/dcrdata/pubsub/v4 v4.0.2
	github.com/decred/dcrdata/rpcutils/v3 v3.0.1
	github.com/decred/dcrdata/semver v1.0.0
	github.com/decred/dcrdata/stakedb/v3 v3.1.1
	github.com/decred/dcrdata/txhelpers v1.1.0
	github.com/decred/dcrdata/txhelpers/v2 v2.0.0
	github.com/decred/dcrdata/v5 v5.2.2
	github.com/decred/slog v1.1.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/google/gops v0.3.13
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/logrotate v1.0.0
	github.com/planetdecred/pdanalytics/web v0.0.0-00010101000000-000000000000
	github.com/prometheus/common v0.15.0
)

replace github.com/planetdecred/pdanalytics/web => ../../web
