module github.com/planetdecred/pdanalytics/pkgs/stakingreward

go 1.13

require (
	github.com/decred/dcrd/chaincfg/v2 v2.3.0
	github.com/decred/dcrd/dcrutil v1.4.0
	github.com/decred/dcrd/rpcclient/v5 v5.0.1
	github.com/decred/dcrd/wire v1.4.0
	github.com/decred/dcrdata/exchanges/v2 v2.1.0
	github.com/decred/slog v1.1.0
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/planetdecred/pdanalytics/web v0.0.0-20210121232737-d068a16f7d67
	golang.org/x/text v0.3.2 // indirect
)

replace github.com/planetdecred/pdanalytics/web => ../../web
