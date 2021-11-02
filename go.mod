module github.com/planetdecred/pdanalytics

go 1.13

require (
	github.com/asdine/storm/v3 v3.2.1
	github.com/caarlos0/env v3.5.0+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/decred/dcrd/blockchain/stake/v4 v4.0.0-20211030183058-7368e0d79182
	github.com/decred/dcrd/chaincfg/chainhash v1.0.4-0.20210914212651-723d86274b0d
	github.com/decred/dcrd/chaincfg/v2 v2.3.0
	github.com/decred/dcrd/dcrutil v1.4.0
	github.com/decred/dcrd/dcrutil/v2 v2.0.1
	github.com/decred/dcrd/dcrutil/v4 v4.0.0-20210925154931-7b184ab3fd61
	github.com/decred/dcrd/peer/v2 v2.2.0
	github.com/decred/dcrd/rpc/jsonrpc/types/v2 v2.3.0
	github.com/decred/dcrd/rpcclient/v5 v5.0.1
	github.com/decred/dcrd/wire v1.4.1-0.20210914212651-723d86274b0d
	github.com/decred/dcrdata/exchanges/v2 v2.1.0
	github.com/decred/dcrdata/gov/v4 v4.0.0
	github.com/decred/dcrdata/v5 v5.2.2
	github.com/decred/dcrdata/v6 v6.0.0
	github.com/decred/dcrdata/v7 v7.0.0-20211005214036-90e57d284420
	github.com/decred/politeia v1.2.0
	github.com/decred/slog v1.2.0
	github.com/friendsofgo/errors v0.9.2
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/google/gops v0.3.13
	github.com/jessevdk/go-flags v1.4.1-0.20200711081900-c17162fe8fd7
	github.com/jrick/logrotate v1.0.0
	github.com/kat-co/vala v0.0.0-20170210184112-42e1d8b61f12
	github.com/lib/pq v1.9.0
	github.com/planetdecred/pdanalytics/dcrd v0.0.0-00010101000000-000000000000
	github.com/planetdecred/pdanalytics/web v0.0.0-20210121232737-d068a16f7d67
	github.com/spf13/viper v1.7.1
	github.com/volatiletech/null/v8 v8.1.2
	github.com/volatiletech/randomize v0.0.1
	github.com/volatiletech/sqlboiler/v4 v4.5.0
	github.com/volatiletech/strmangle v0.0.1
	google.golang.org/genproto v0.0.0-20201022181438-0ff5f38871d5 // indirect
	google.golang.org/grpc v1.36.1 // indirect
	gopkg.in/ini.v1 v1.55.0 // indirect
)

replace (
	github.com/planetdecred/pdanalytics/dcrd => ./dcrd
	github.com/planetdecred/pdanalytics/web => ./web
)
