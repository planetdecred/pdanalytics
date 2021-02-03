# pdanalytics


## Instalation

`npm install`
`npm run build`

## Building
go build -o pdanalytics -v -ldflags \
    "-X github.com/planetdecred/pdanalytics/version.appPreRelease=beta \
     -X github.com/planetdecred/pdanalytics/version.appBuild=`git rev-parse --short HEAD`"