module github.com/planetdecred/pdanalytics

go 1.13

require (
	github.com/DATA-DOG/go-sqlmock v1.5.0 // indirect
	github.com/asdine/storm/v3 v3.2.1
	github.com/caarlos0/env v3.5.0+incompatible
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/chaincfg/chainhash v1.0.2
	github.com/decred/dcrd/chaincfg/v2 v2.3.0
	github.com/decred/dcrd/chaincfg/v3 v3.0.0
	github.com/decred/dcrd/dcrjson/v3 v3.1.0
	github.com/decred/dcrd/dcrutil v1.4.0
	github.com/decred/dcrd/dcrutil/v2 v2.0.1
	github.com/decred/dcrd/gcs/v2 v2.1.0 // indirect
	github.com/decred/dcrd/peer/v2 v2.2.0
	github.com/decred/dcrd/rpc/jsonrpc/types/v2 v2.3.0
	github.com/decred/dcrd/rpcclient/v5 v5.0.1
	github.com/decred/dcrd/wire v1.4.0
	github.com/decred/dcrdata/exchanges/v2 v2.1.0
	github.com/decred/dcrdata/v5 v5.2.2
	github.com/decred/politeia v0.0.0-20191031182202-b33af07598f2
	github.com/decred/slog v1.1.0
	github.com/dmigwi/go-piparser/proposals v0.0.0-20191219171828-ae8cbf4067e1
	github.com/friendsofgo/errors v0.9.2
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/google/gops v0.3.13
	github.com/jessevdk/go-flags v1.4.0
	github.com/jrick/logrotate v1.0.0
	github.com/kat-co/vala v0.0.0-20170210184112-42e1d8b61f12
	github.com/kr/pretty v0.2.0 // indirect
	github.com/lib/pq v1.2.1-0.20191011153232-f91d3411e481
	github.com/planetdecred/pdanalytics/dcrd v0.0.0-00010101000000-000000000000
	github.com/planetdecred/pdanalytics/web v0.0.0-20210121232737-d068a16f7d67
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.4.0 // indirect
	github.com/volatiletech/null/v8 v8.1.2
	github.com/volatiletech/randomize v0.0.1
	github.com/volatiletech/sqlboiler/v4 v4.5.0
	github.com/volatiletech/strmangle v0.0.1
	golang.org/x/crypto v0.0.0-20190923035154-9ee001bba392 // indirect
)

replace (
	github.com/planetdecred/pdanalytics/dcrd => ./dcrd
	github.com/planetdecred/pdanalytics/web => ./web
)
