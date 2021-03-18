# pdanalytics


## Instalation

`npm install`
`npm run build`

## Linting
golangci-lint run --deadline=10m --disable-all \
    --enable govet \
    --enable staticcheck \
    --enable gosimple \
    --enable unconvert \
    --enable ineffassign \
    --enable goimports \
    --enable misspell

## Building
go build -o pdanalytics -v -ldflags \
    "-X github.com/planetdecred/pdanalytics/version.appPreRelease=beta \
     -X github.com/planetdecred/pdanalytics/version.appBuild=`git rev-parse --short HEAD`"