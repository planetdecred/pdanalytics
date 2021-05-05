package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/decred/slog"
	"github.com/jrick/logrotate/rotator"
	"github.com/planetdecred/pdanalytics/attackcost"
	"github.com/planetdecred/pdanalytics/chart"
	"github.com/planetdecred/pdanalytics/charts"
	"github.com/planetdecred/pdanalytics/gov/politeia"
	"github.com/planetdecred/pdanalytics/homepage"
	"github.com/planetdecred/pdanalytics/mempool"
	"github.com/planetdecred/pdanalytics/netsnapshot"
	"github.com/planetdecred/pdanalytics/parameters"
	"github.com/planetdecred/pdanalytics/postgres"
	"github.com/planetdecred/pdanalytics/propagation"
	"github.com/planetdecred/pdanalytics/stakingreward"
	"github.com/planetdecred/pdanalytics/web"
)

// logWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type logWriter struct{}

// Write writes the data in p to standard out and the log rotator.
func (logWriter) Write(p []byte) (n int, err error) {
	os.Stdout.Write(p)
	return logRotator.Write(p)
}

// Loggers per subsystem.  A single backend logger is created and all subsytem
// loggers created from it will write to the backend.  When adding new
// subsystems, add the subsystem logger variable here and to the
// subsystemLoggers map.
//
// Loggers can not be used before the log rotator has been initialized with a
// log file.  This must be performed early during application startup by calling
// initLogRotator.
var (
	// backendLog is the logging backend used to create all subsystem loggers.
	// The backend must not be used before the log rotator has been initialized,
	// or data races and/or nil pointer dereferences will occur.
	backendLog = slog.NewBackend(logWriter{})

	// logRotator is one of the logging outputs.  It should be closed on
	// application shutdown.
	logRotator *rotator.Rotator

	log              = backendLog.Logger("PDAN")
	paramLog         = backendLog.Logger("PARA")
	attackcostLog    = backendLog.Logger("ATCK")
	stakingrewardLog = backendLog.Logger("STCK")
	homeLog          = backendLog.Logger(("HOME"))
	psqlLog          = backendLog.Logger("PSQL")
	mempoolLog       = backendLog.Logger("MEMP")
	chartLog         = backendLog.Logger("CHRT")
	propLog          = backendLog.Logger("PROP")
	snapshotLog      = backendLog.Logger("NETS")
	politeiaLog      = backendLog.Logger("POLI")
	webLogger        = backendLog.Logger("WEBL")
	chartsLog        = backendLog.Logger("CHRTS")
)

// Initialize package-global logger variables.
func init() {
	parameters.UseLogger(paramLog)
	stakingreward.UseLogger(stakingrewardLog)
	attackcost.UseLogger(attackcostLog)
	homepage.UseLogger(homeLog)
	mempool.UseLogger(mempoolLog)
	postgres.UseLogger(psqlLog)
	chart.UseLogger(chartLog)
	propagation.UseLogger(propLog)
	netsnapshot.UseLogger(snapshotLog)
	politeia.UseLogger(politeiaLog)
	web.UseLogger(webLogger)
	charts.UseLogger(chartsLog)
}

// subsystemLoggers maps each subsystem identifier to its associated logger.
var subsystemLoggers = map[string]slog.Logger{
	"PDAN":  log,
	"PARA":  paramLog,
	"ATCK":  attackcostLog,
	"STCK":  stakingrewardLog,
	"HOME":  homeLog,
	"PROP":  propLog,
	"NETS":  snapshotLog,
	"WEBL":  webLogger,
	"MEMP":  mempoolLog,
	"PSQL":  psqlLog,
	"POLI":  politeiaLog,
	"CHRTS": chartsLog,
}

// initLogRotator initializes the logging rotater to write logs to logFile and
// create roll files in the same directory.  It must be called before the
// package-global log rotater variables are used.
func initLogRotator(logFile string, maxRolls int) {
	logDir, _ := filepath.Split(logFile)
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(1)
	}
	r, err := rotator.New(logFile, 32*1024, false, maxRolls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %v\n", err)
		os.Exit(1)
	}

	logRotator = r
}

// setLogLevel sets the logging level for provided subsystem.  Invalid
// subsystems are ignored.  Uninitialized subsystems are dynamically created as
// needed.
func setLogLevel(subsystemID string, logLevel string) {
	// Ignore invalid subsystems.
	logger, ok := subsystemLoggers[subsystemID]
	if !ok {
		return
	}

	// Defaults to info if the log level is invalid.
	level, _ := slog.LevelFromString(logLevel)
	logger.SetLevel(level)
}

// setLogLevels sets the log level for all subsystem loggers to the passed
// level.  It also dynamically creates the subsystem loggers as needed, so it
// can be used to initialize the logging system.
func setLogLevels(logLevel string) {
	// Configure all sub-systems with the new logging level.  Dynamically
	// create loggers as needed.
	for subsystemID := range subsystemLoggers {
		setLogLevel(subsystemID, logLevel)
	}
}
