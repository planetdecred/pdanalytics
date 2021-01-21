module github.com/planetdecred/pdanalytics/pkgs/parameters

go 1.13

require (
	github.com/decred/dcrd/chaincfg/v2 v2.3.0
	github.com/decred/dcrd/rpcclient/v5 v5.0.0
	github.com/decred/dcrd/txscript/v2 v2.1.0
	github.com/decred/slog v1.1.0
	github.com/planetdecred/pdanalytics/web v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.0.0-20191028085509-fe3aa8a45271 // indirect
)

replace github.com/planetdecred/pdanalytics/web => ../../web
