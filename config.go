// Copyright (c) 2016-2019 The Decred developers
// Copyright (c) 2017 Jonathan Chappelow
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/caarlos0/env"
	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/decred/dcrd/dcrutil/v2"
	"github.com/decred/dcrdata/v5/netparams"
	"github.com/decred/dcrdata/v5/version"
	"github.com/decred/slog"
	flags "github.com/jessevdk/go-flags"
)

const (
	defaultConfigFilename = "pdanalytics.conf"
	defaultLogFilename    = "pdanalytics.log"
	defaultDataDirname    = "data"
	defaultLogLevel       = "info"
	defaultLogDirname     = "logs"
)

var activeNet = &netparams.MainNetParams
var activeChain = chaincfg.MainNetParams()

var (
	defaultHomeDir           = dcrutil.AppDataDir("pdanalytics", false)
	defaultConfigFile        = filepath.Join(defaultHomeDir, defaultConfigFilename)
	defaultLogDir            = filepath.Join(defaultHomeDir, defaultLogDirname)
	defaultDataDir           = filepath.Join(defaultHomeDir, defaultDataDirname)
	dcrdHomeDir              = dcrutil.AppDataDir("dcrd", false)
	defaultDaemonRPCCertFile = filepath.Join(dcrdHomeDir, "rpc.cert")
	defaultMaxLogZips        = 16

	defaultHost         = "localhost"
	defaultHTTPProfPath = "/p"
	defaultAPIProto     = "http"
	defaultMainnetPort  = "7777"
	defaultTestnetPort  = "17778"
	defaultSimnetPort   = "17779"
	defaultServerHeader = "pdanalytics"

	defaultMainnetLink  = "https://explorer.dcrdata.org/"
	defaultTestnetLink  = "https://testnet.dcrdata.org/"
	defaultOnionAddress = ""
)

type config struct {
	// General application behavior
	HomeDir      string `short:"A" long:"appdata" description:"Path to application home directory" env:"DCRDATA_APPDATA_DIR"`
	ConfigFile   string `short:"C" long:"configfile" description:"Path to configuration file" env:"DCRDATA_CONFIG_FILE"`
	DataDir      string `short:"b" long:"datadir" description:"Directory to store data" env:"DCRDATA_DATA_DIR"`
	LogDir       string `long:"logdir" description:"Directory to log output." env:"DCRDATA_LOG_DIR"`
	MaxLogZips   int    `long:"max-log-zips" description:"The number of zipped log files created by the log rotator to be retained. Setting to 0 will keep all."`
	OutFolder    string `short:"f" long:"outfolder" description:"Folder for file outputs" env:"DCRDATA_OUT_FOLDER"`
	ShowVersion  bool   `short:"V" long:"version" description:"Display version information and exit"`
	TestNet      bool   `long:"testnet" description:"Use the test network (default mainnet)" env:"DCRDATA_USE_TESTNET"`
	SimNet       bool   `long:"simnet" description:"Use the simulation test network (default mainnet)" env:"DCRDATA_USE_SIMNET"`
	DebugLevel   string `short:"d" long:"debuglevel" description:"Logging level {trace, debug, info, warn, error, critical}" env:"DCRDATA_LOG_LEVEL"`
	Quiet        bool   `short:"q" long:"quiet" description:"Easy way to set debuglevel to error" env:"DCRDATA_QUIET"`
	HTTPProfile  bool   `long:"httpprof" short:"p" description:"Start HTTP profiler." env:"DCRDATA_ENABLE_HTTP_PROFILER"`
	HTTPProfPath string `long:"httpprofprefix" description:"URL path prefix for the HTTP profiler." env:"DCRDATA_HTTP_PROFILER_PREFIX"`
	CPUProfile   string `long:"cpuprofile" description:"File for CPU profiling." env:"DCRDATA_CPU_PROFILER_FILE"`
	UseGops      bool   `short:"g" long:"gops" description:"Run with gops diagnostics agent listening. See github.com/google/gops for more information." env:"DCRDATA_USE_GOPS"`

	// API/server
	APIProto     string `long:"apiproto" description:"Protocol for API (http or https)" env:"DCRDATA_ENABLE_HTTPS"`
	APIListen    string `long:"apilisten" description:"Listen address for API. default localhost:7777, :17778 testnet, :17779 simnet" env:"DCRDATA_LISTEN_URL"`
	ServerHeader string `long:"server-http-header" description:"Set the HTTP response header Server key value. Valid values are \"off\", \"version\", or a custom string."`

	// RPC client options
	DcrdUser         string `long:"dcrduser" description:"Daemon RPC user name" env:"DCRDATA_DCRD_USER"`
	DcrdPass         string `long:"dcrdpass" description:"Daemon RPC password" env:"DCRDATA_DCRD_PASS"`
	DcrdServ         string `long:"dcrdserv" description:"Hostname/IP and port of dcrd RPC server to connect to (default localhost:9109, testnet: localhost:19109, simnet: localhost:19556)" env:"DCRDATA_DCRD_URL"`
	DcrdCert         string `long:"dcrdcert" description:"File containing the dcrd certificate file" env:"DCRDATA_DCRD_CERT"`
	DisableDaemonTLS bool   `long:"nodaemontls" description:"Disable TLS for the daemon RPC client -- NOTE: This is only allowed if the RPC client is connecting to localhost" env:"DCRDATA_DCRD_DISABLE_TLS"`

	// Links
	MainnetLink  string `long:"mainnet-link" description:"When dcrdata is on testnet, this address will be used to direct a user to a dcrdata on mainnet when appropriate." env:"DCRDATA_MAINNET_LINK"`
	TestnetLink  string `long:"testnet-link" description:"When dcrdata is on mainnet, this address will be used to direct a user to a dcrdata on testnet when appropriate." env:"DCRDATA_TESTNET_LINK"`
	OnionAddress string `long:"onion-address" description:"Hidden service address" env:"DCRDATA_ONION_ADDRESS"`
}

var (
	defaultConfig = config{
		HomeDir:      defaultHomeDir,
		DataDir:      defaultDataDir,
		LogDir:       defaultLogDir,
		MaxLogZips:   defaultMaxLogZips,
		ConfigFile:   defaultConfigFile,
		DebugLevel:   defaultLogLevel,
		HTTPProfPath: defaultHTTPProfPath,
		APIProto:     defaultAPIProto,
		ServerHeader: defaultServerHeader,
		DcrdCert:     defaultDaemonRPCCertFile,
		MainnetLink:  defaultMainnetLink,
		TestnetLink:  defaultTestnetLink,
		OnionAddress: defaultOnionAddress,
	}
)

// cleanAndExpandPath expands environment variables and leading ~ in the passed
// path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// NOTE: The os.ExpandEnv doesn't work with Windows cmd.exe-style
	// %VARIABLE%, but the variables can still be expanded via POSIX-style
	// $VARIABLE.
	path = os.ExpandEnv(path)

	if !strings.HasPrefix(path, "~") {
		return filepath.Clean(path)
	}

	// Expand initial ~ to the current user's home directory, or ~otheruser to
	// otheruser's home directory.  On Windows, both forward and backward
	// slashes can be used.
	path = path[1:]

	var pathSeparators string
	if runtime.GOOS == "windows" {
		pathSeparators = string(os.PathSeparator) + "/"
	} else {
		pathSeparators = string(os.PathSeparator)
	}

	userName := ""
	if i := strings.IndexAny(path, pathSeparators); i != -1 {
		userName = path[:i]
		path = path[i:]
	}

	homeDir := ""
	var u *user.User
	var err error
	if userName == "" {
		u, err = user.Current()
	} else {
		u, err = user.Lookup(userName)
	}
	if err == nil {
		homeDir = u.HomeDir
	}
	// Fallback to CWD if user lookup fails or user has no home directory.
	if homeDir == "" {
		homeDir = "."
	}

	return filepath.Join(homeDir, path)
}

// normalizeNetworkAddress checks for a valid local network address format and
// adds default host and port if not present. Invalidates addresses that include
// a protocol identifier.
func normalizeNetworkAddress(a, defaultHost, defaultPort string) (string, error) {
	if strings.Contains(a, "://") {
		return a, fmt.Errorf("Address %s contains a protocol identifier, which is not allowed", a)
	}
	if a == "" {
		return defaultHost + ":" + defaultPort, nil
	}
	host, port, err := net.SplitHostPort(a)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") {
			normalized := a + ":" + defaultPort
			host, port, err = net.SplitHostPort(normalized)
			if err != nil {
				return a, fmt.Errorf("Unable to address %s after port resolution: %v", normalized, err)
			}
		} else {
			return a, fmt.Errorf("Unable to normalize address %s: %v", a, err)
		}
	}
	if host == "" {
		host = defaultHost
	}
	if port == "" {
		port = defaultPort
	}
	return host + ":" + port, nil
}

// validLogLevel returns whether or not logLevel is a valid debug log level.
func validLogLevel(logLevel string) bool {
	_, ok := slog.LevelFromString(logLevel)
	return ok
}

// parseAndSetDebugLevels attempts to parse the specified debug level and set
// the levels accordingly.  An appropriate error is returned if anything is
// invalid.
func parseAndSetDebugLevels(debugLevel string) error {
	// When the specified string doesn't have any delimiters, treat it as
	// the log level for all subsystems.
	if !strings.Contains(debugLevel, ",") && !strings.Contains(debugLevel, "=") {
		// Validate debug log level.
		if !validLogLevel(debugLevel) {
			str := "The specified debug level [%v] is invalid"
			return fmt.Errorf(str, debugLevel)
		}

		// Change the logging level for all subsystems.
		setLogLevels(debugLevel)

		return nil
	}

	// Split the specified string into subsystem/level pairs while detecting
	// issues and update the log levels accordingly.
	for _, logLevelPair := range strings.Split(debugLevel, ",") {
		if !strings.Contains(logLevelPair, "=") {
			str := "The specified debug level contains an invalid " +
				"subsystem/level pair [%v]"
			return fmt.Errorf(str, logLevelPair)
		}

		// Extract the specified subsystem and log level.
		fields := strings.Split(logLevelPair, "=")
		subsysID, logLevel := fields[0], fields[1]

		// Validate log level.
		if !validLogLevel(logLevel) {
			str := "The specified debug level [%v] is invalid"
			return fmt.Errorf(str, logLevel)
		}

		setLogLevel(subsysID, logLevel)
	}

	return nil
}

// loadConfig initializes and parses the config using a config file and command
// line options.
func loadConfig() (*config, error) {
	loadConfigError := func(err error) (*config, error) {
		return nil, err
	}

	// Default config
	cfg := defaultConfig
	defaultConfigNow := defaultConfig

	// Load settings from environment variables.
	err := env.Parse(&cfg)
	if err != nil {
		return loadConfigError(err)
	}

	// If appdata was specified but not the config file, change the config file
	// path, and record this as the new default config file location.
	if defaultHomeDir != cfg.HomeDir && defaultConfigNow.ConfigFile == cfg.ConfigFile {
		cfg.ConfigFile = filepath.Join(cfg.HomeDir, defaultConfigFilename)
		// Update the defaultConfig to avoid an error if the config file in this
		// "new default" location does not exist.
		defaultConfigNow.ConfigFile = cfg.ConfigFile
	}

	// Pre-parse the command line options to see if an alternative config file
	// or the version flag was specified. Override any environment variables
	// with parsed command line flags.
	preCfg := cfg
	preParser := flags.NewParser(&preCfg, flags.HelpFlag|flags.PassDoubleDash)
	_, flagerr := preParser.Parse()

	if flagerr != nil {
		e, ok := flagerr.(*flags.Error)
		if !ok || e.Type != flags.ErrHelp {
			preParser.WriteHelp(os.Stderr)
		}
		if ok && e.Type == flags.ErrHelp {
			preParser.WriteHelp(os.Stdout)
			os.Exit(0)
		}
		return loadConfigError(flagerr)
	}

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	if preCfg.ShowVersion {
		fmt.Printf("%s version %s (Go version %s)\n", appName,
			version.Version(), runtime.Version())
		os.Exit(0)
	}

	// If a non-default appdata folder is specified on the command line, it may
	// be necessary adjust the config file location. If the the config file
	// location was not specified on the command line, the default location
	// should be under the non-default appdata directory. However, if the config
	// file was specified on the command line, it should be used regardless of
	// the appdata directory.
	if defaultHomeDir != preCfg.HomeDir && defaultConfigNow.ConfigFile == preCfg.ConfigFile {
		preCfg.ConfigFile = filepath.Join(preCfg.HomeDir, defaultConfigFilename)
		// Update the defaultConfig to avoid an error if the config file in this
		// "new default" location does not exist.
		defaultConfigNow.ConfigFile = preCfg.ConfigFile
	}

	// Load additional config from file.
	var configFileError error
	// Config file name for logging.
	configFile := "NONE (defaults)"
	parser := flags.NewParser(&cfg, flags.Default)

	// Do not error default config file is missing.
	if _, err := os.Stat(preCfg.ConfigFile); os.IsNotExist(err) {
		// Non-default config file must exist
		if defaultConfigNow.ConfigFile != preCfg.ConfigFile {
			fmt.Fprintln(os.Stderr, err)
			return loadConfigError(err)
		}
		// Warn about missing default config file, but continue
		fmt.Printf("Config file (%s) does not exist. Using defaults.\n",
			preCfg.ConfigFile)
	} else {
		// The config file exists, so attempt to parse it.
		err = flags.NewIniParser(parser).ParseFile(preCfg.ConfigFile)
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				fmt.Fprintln(os.Stderr, err)
				parser.WriteHelp(os.Stderr)
				return loadConfigError(err)
			}
			configFileError = err
		}
		configFile = preCfg.ConfigFile
	}

	// Parse command line options again to ensure they take precedence.
	_, err = parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			parser.WriteHelp(os.Stderr)
		}
		return loadConfigError(err)
	}

	// Create the home directory if it doesn't already exist.
	funcName := "loadConfig"
	err = os.MkdirAll(cfg.HomeDir, 0700)
	if err != nil {
		// Show a nicer error message if it's because a symlink is linked to a
		// directory that does not exist (probably because it's not mounted).
		if e, ok := err.(*os.PathError); ok && os.IsExist(err) {
			if link, lerr := os.Readlink(e.Path); lerr == nil {
				str := "is symlink %s -> %s mounted?"
				err = fmt.Errorf(str, e.Path, link)
			}
		}

		str := "%s: failed to create home directory: %v"
		err := fmt.Errorf(str, funcName, err)
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}

	// If a non-default appdata folder is specified, it may be necessary to
	// adjust the DataDir and LogDir. If these other paths are their defaults,
	// they should be modifed to look under the non-default appdata directory.
	// If they are not their defaults, the user-specified values should be used.
	if defaultHomeDir != cfg.HomeDir {
		if defaultDataDir == cfg.DataDir {
			cfg.DataDir = filepath.Join(cfg.HomeDir, defaultDataDirname)
		}
		if defaultLogDir == cfg.LogDir {
			cfg.LogDir = filepath.Join(cfg.HomeDir, defaultLogDirname)
		}
	}

	// Warn about missing config file after the final command line parse
	// succeeds.  This prevents the warning on help messages and invalid
	// options.
	if configFileError != nil {
		fmt.Printf("%v\n", configFileError)
		return loadConfigError(configFileError)
	}

	// Choose the active network params based on the selected network. Multiple
	// networks can't be selected simultaneously.
	numNets := 0
	activeNet = &netparams.MainNetParams
	activeChain = chaincfg.MainNetParams()
	defaultPort := defaultMainnetPort
	if cfg.TestNet {
		activeNet = &netparams.TestNet3Params
		activeChain = chaincfg.TestNet3Params()
		defaultPort = defaultTestnetPort
		numNets++
	}
	if cfg.SimNet {
		activeNet = &netparams.SimNetParams
		activeChain = chaincfg.SimNetParams()
		defaultPort = defaultSimnetPort
		numNets++
	}
	if numNets > 1 {
		str := "%s: the testnet and simnet params can't be " +
			"used together -- choose one of the three"
		err := fmt.Errorf(str, funcName)
		fmt.Fprintln(os.Stderr, err)
		parser.WriteHelp(os.Stderr)
		return loadConfigError(err)
	}

	// Append the network type to the data directory so it is "namespaced" per
	// network.  In addition to the block database, there are other pieces of
	// data that are saved to disk such as address manager state. All data is
	// specific to a network, so namespacing the data directory means each
	// individual piece of serialized data does not have to worry about changing
	// names per network and such.
	//
	// Make list of old versions of testnet directories here since the network
	// specific DataDir will be used after this.
	cfg.DataDir = cleanAndExpandPath(cfg.DataDir)
	cfg.DataDir = filepath.Join(cfg.DataDir, activeNet.Name)
	// Create the data folder if it does not exist.
	err = os.MkdirAll(cfg.DataDir, 0700)
	if err != nil {
		return nil, err
	}

	logRotator = nil
	// Append the network type to the log directory so it is "namespaced"
	// per network in the same fashion as the data directory.
	cfg.LogDir = cleanAndExpandPath(cfg.LogDir)
	cfg.LogDir = filepath.Join(cfg.LogDir, activeNet.Name)

	// Initialize log rotation. After log rotation has been initialized, the
	// logger variables may be used. This creates the LogDir if needed.
	if cfg.MaxLogZips < 0 {
		cfg.MaxLogZips = 0
	}
	initLogRotator(filepath.Join(cfg.LogDir, defaultLogFilename), cfg.MaxLogZips)

	log.Infof("Log folder:  %s", cfg.LogDir)
	log.Infof("Config file: %s", configFile)

	// Set the host names and ports to the default if the user does not specify
	// them.
	cfg.DcrdServ, err = normalizeNetworkAddress(cfg.DcrdServ, defaultHost, activeNet.JSONRPCClientPort)
	if err != nil {
		return loadConfigError(err)
	}

	// Output folder
	cfg.OutFolder = cleanAndExpandPath(cfg.OutFolder)
	cfg.OutFolder = filepath.Join(cfg.OutFolder, activeNet.Name)

	// Ensure HTTP profiler is mounted with a valid path prefix.
	if cfg.HTTPProfile && (cfg.HTTPProfPath == "/" || len(defaultHTTPProfPath) == 0) {
		return loadConfigError(fmt.Errorf("httpprofprefix must not be \"\" or \"/\""))
	}

	// Parse, validate, and set debug log level(s).
	if cfg.Quiet {
		cfg.DebugLevel = "error"
	}

	// Parse, validate, and set debug log level(s).
	if err := parseAndSetDebugLevels(cfg.DebugLevel); err != nil {
		err = fmt.Errorf("%s: %v", funcName, err.Error())
		fmt.Fprintln(os.Stderr, err)
		parser.WriteHelp(os.Stderr)
		return loadConfigError(err)
	}

	// Check the supplied APIListen address
	if cfg.APIListen == "" {
		cfg.APIListen = defaultHost + ":" + defaultPort
	} else {
		cfg.APIListen, err = normalizeNetworkAddress(cfg.APIListen, defaultHost, defaultPort)
		if err != nil {
			return loadConfigError(err)
		}
	}

	switch cfg.ServerHeader {
	case "off":
		cfg.ServerHeader = ""
	case "version":
		cfg.ServerHeader = version.AppName + "-" + version.Version()
	}

	// Expand some additional paths.
	cfg.DcrdCert = cleanAndExpandPath(cfg.DcrdCert)
	// cfg.RateCertificate = cleanAndExpandPath(cfg.RateCertificate)

	// Clean up the provided mainnet and testnet links, ensuring there is a single
	// trailing slash.
	cfg.MainnetLink = strings.TrimSuffix(cfg.MainnetLink, "/") + "/"
	cfg.TestnetLink = strings.TrimSuffix(cfg.TestnetLink, "/") + "/"

	return &cfg, nil
}
