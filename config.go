package main

import (
	"fmt"
	"io"
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
	"github.com/decred/slog"
	flags "github.com/jessevdk/go-flags"
	"github.com/planetdecred/pdanalytics/commstats"
	"github.com/planetdecred/pdanalytics/netsnapshot"
	"github.com/planetdecred/pdanalytics/version"
)

const (
	defaultConfigFilename = "pdanalytics.conf"
	sampleConfigFileName  = "./sample-pdanalytics.conf"
	defaultLogFilename    = "pdanalytics.log"
	defaultDataDirname    = "data"
	defaultLogLevel       = "info"
	defaultLogDirname     = "logs"
	defaultDbHost         = "localhost"
	defaultDbPort         = "5432"
	defaultDbUser         = "postgres"
	defaultDbPass         = "postgres"
	defaultDbName         = "pdanalytics"
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

	defaultHost               = "localhost"
	defaultHTTPProfPath       = "/p"
	defaultAPIProto           = "http"
	defaultMainnetPort        = "7070"
	defaultTestnetPort        = "7171"
	defaultSimnetPort         = "7272"
	defaultCacheControlMaxAge = 86400
	defaultServerHeader       = "pdanalytics"

	// Exchange bot
	defaultExchangeIndex     = "USD"
	defaultDisabledExchanges = "dragonex,poloniex"
	defaultRateCertFile      = filepath.Join(defaultHomeDir, "rpc.cert")

	defaultMainnetLink  = "https://explorer.planetdecred.org/"
	defaultTestnetLink  = "https://testnet.planetdecred.org/"
	defaultOnionAddress = ""
	defaultAPIURL       = "https://explorer.planetdecred.org/api/"

	defaultMempoolInterval = 60.0
	defaultPowInterval     = 300
	defaultVSPInterval     = 300

	// network snapshot
	defaultSnapshotInterval  = 720
	defaultSeeder            = "127.0.0.1"
	defaultSeederPort        = 9108
	maxPeerConnectionFailure = 3

	// gov
	defaultAgendasDBFileName  = "agendas.db"
	defaultProposalsFileName  = filepath.Join(defaultDataDir, "proposals.db")
	defaultPoliteiaAPIURl     = "https://proposals.decred.org/"
	defaultPiPropoalRepoOwner = "decred-proposals"
	defaultPiProposalRepo     = "mainnet"

	// community
	defaultRedditInterval      = 60
	defaultTwitterStatInterval = 60 * 24
	defaultGithubStatInterval  = 60 * 24
	defaultYoutubeInterval     = 60 * 24
	defaultSubreddits          = []string{"decred"}
	defaultTwitterHandles      = []string{"decredproject"}
	defaultGithubRepositories  = []string{"decred/dcrd", "decred/dcrdata", "decred/dcrwallet", "decred/politeia", "decred/decrediton"}
	defaultYoutubeChannelNames = []string{"Decred"}
	defaultYoutubeChannelId    = []string{"UCJ2bYDaPYHpSmJPh_M5dNSg"}
)

type config struct {
	// RPC client options
	DcrdRPCUser      string `long:"dcrduser" description:"Daemon RPC user name" env:"PDANALYTICS_DCRD_USER"`
	DcrdRPCPassword  string `long:"dcrdpass" description:"Daemon RPC password" env:"PDANALYTICS_DCRD_PASS"`
	DcrdRPCServer    string `long:"dcrdserv" description:"Hostname/IP and port of dcrd RPC server to connect to (default localhost:9109, testnet: localhost:19109, simnet: localhost:19556)" env:"PDANALYTICS_DCRD_URL"`
	DcrdCert         string `long:"dcrdcert" description:"File containing the dcrd certificate file" env:"PDANALYTICS_DCRD_CERT"`
	DisableDaemonTLS bool   `long:"nodaemontls" description:"Disable TLS for the daemon RPC client -- NOTE: This is only allowed if the RPC client is connecting to localhost" env:"PDANALYTICS_DCRD_DISABLE_TLS"`

	// General application behavior
	HomeDir      string `short:"A" long:"appdata" description:"Path to application home directory" env:"PDANALYTICS_APPDATA_DIR"`
	ConfigFile   string `short:"C" long:"configfile" description:"Path to configuration file" env:"PDANALYTICS_CONFIG_FILE"`
	DataDir      string `short:"b" long:"datadir" description:"Directory to store data" env:"PDANALYTICS_DATA_DIR"`
	LogDir       string `long:"logdir" description:"Directory to log output." env:"PDANALYTICS_LOG_DIR"`
	MaxLogZips   int    `long:"max-log-zips" description:"The number of zipped log files created by the log rotator to be retained. Setting to 0 will keep all."`
	OutFolder    string `short:"f" long:"outfolder" description:"Folder for file outputs" env:"PDANALYTICS_OUT_FOLDER"`
	ShowVersion  bool   `short:"V" long:"version" description:"Display version information and exit"`
	TestNet      bool   `long:"testnet" description:"Use the test network (default mainnet)" env:"PDANALYTICS_USE_TESTNET"`
	SimNet       bool   `long:"simnet" description:"Use the simulation test network (default mainnet)" env:"PDANALYTICS_USE_SIMNET"`
	DebugLevel   string `short:"d" long:"debuglevel" description:"Logging level {trace, debug, info, warn, error, critical}" env:"PDANALYTICS_LOG_LEVEL"`
	Quiet        bool   `short:"q" long:"quiet" description:"Easy way to set debuglevel to error" env:"PDANALYTICS_QUIET"`
	HTTPProfile  bool   `long:"httpprof" short:"p" description:"Start HTTP profiler." env:"PDANALYTICS_ENABLE_HTTP_PROFILER"`
	HTTPProfPath string `long:"httpprofprefix" description:"URL path prefix for the HTTP profiler." env:"PDANALYTICS_HTTP_PROFILER_PREFIX"`
	CPUProfile   string `long:"cpuprofile" description:"File for CPU profiling." env:"PDANALYTICS_CPU_PROFILER_FILE"`
	UseGops      bool   `short:"g" long:"gops" description:"Run with gops diagnostics agent listening. See github.com/google/gops for more information." env:"PDANALYTICS_USE_GOPS"`
	ReloadHTML   bool   `long:"reload-html" description:"Reload HTML templates on every request" env:"DCRDATA_RELOAD_HTML"`
	NoHttp       bool   `long:"nohttp" description:"Disables http server from running"`
	APIURL       string `long:"apiurl" description:"Base API URL where pdanalytics will pull data from"`

	// Postgresql Configuration
	DBHost string `long:"dbhost" description:"Database host"`
	DBPort string `long:"dbport" description:"Database port"`
	DBUser string `long:"dbuser" description:"Database username"`
	DBPass string `long:"dbpass" description:"Database password"`
	DBName string `long:"dbname" description:"Database name"`

	// API/server
	APIProto           string `long:"apiproto" description:"Protocol for API (http or https)" env:"PDANALYTICS_ENABLE_HTTPS"`
	APIListen          string `long:"apilisten" description:"Listen address for API. default localhost:7777, :17778 testnet, :17779 simnet" env:"PDANALYTICS_LISTEN_URL"`
	ServerHeader       string `long:"server-http-header" description:"Set the HTTP response header Server key value. Valid values are \"off\", \"version\", or a custom string."`
	CacheControlMaxAge int    `long:"cachecontrol-maxage" description:"Set CacheControl in the HTTP response header to a value in seconds for clients to cache the response. This applies only to FileServer routes." env:"DCRDATA_MAX_CACHE_AGE"`

	// Politeia/proposals and consensus agendas
	AgendasDBFileName string `long:"agendadbfile" description:"Agendas DB file name (default is agendas.db)." env:"DCRDATA_AGENDAS_DB_FILE_NAME"`
	ProposalsFileName string `long:"proposalsdbfile" description:"Proposals DB file name (default is proposals.db)." env:"DCRDATA_PROPOSALS_DB_FILE_NAME"`
	PoliteiaAPIURL    string `long:"politeiaurl" description:"Defines the root API politeia URL (defaults to https://proposals.decred.org)."`
	PiPropRepoOwner   string `long:"piproposalsowner" description:"Defines the owner to the github repo where Politeia's proposals are pushed."`
	PiPropRepoName    string `long:"piproposalsrepo" description:"Defines the name of the github repo where Politeia's proposals are pushed."`

	// Links
	MainnetLink  string `long:"mainnet-link" description:"When pdanalytics is on testnet, this address will be used to direct a user to a pdanalytics on mainnet when appropriate." env:"PDANALYTICS_MAINNET_LINK"`
	TestnetLink  string `long:"testnet-link" description:"When pdanalytics is on mainnet, this address will be used to direct a user to a pdanalytics on testnet when appropriate." env:"PDANALYTICS_TESTNET_LINK"`
	OnionAddress string `long:"onion-address" description:"Hidden service address" env:"PDANALYTICS_ONION_ADDRESS"`

	// ExchangeBot settings
	EnableExchangeBot bool   `long:"exchange-monitor" description:"Enable the exchange monitor" env:"DCRDATA_MONITOR_EXCHANGES"`
	DisabledExchanges string `long:"disabled-exchanges" description:"Exchanges to disable. See /exchanges/exchanges.go for available exchanges. Use a comma to separate multiple exchanges" env:"DCRDATA_DISABLE_EXCHANGES"`
	ExchangeCurrency  string `long:"exchange-currency" description:"The default bitcoin price index. A 3-letter currency code" env:"DCRDATA_EXCHANGE_INDEX"`
	RateMaster        string `long:"ratemaster" description:"The address of a DCRRates instance. Exchange monitoring will get all data from a DCRRates subscription." env:"DCRDATA_RATE_MASTER"`
	RateCertificate   string `long:"ratecert" description:"File containing DCRRates TLS certificate file." env:"DCRDATA_RATE_MASTER"`

	// Modules config
	EnableChainParameters         bool `long:"parameters" description:"Enable/Disables the chain parameter component."`
	EnableAttackCost              bool `long:"attack-cost" description:"Enable/Disables the attack cost calculator component."`
	EnableStakingRewardCalculator bool `long:"staking-reward" description:"Enable/Disables the staking reward calculator component."`
	EnableMempool                 bool `long:"mempool" description:"Enable/Disables the mempool component from running."`
	EnablePropagation             bool `long:"propagation" description:"Enable/Disable the propagation module from running"`
	EnableProposals               bool `long:"proposals" description:"Enable/Disable the proposals module from running"`
	EnableProposalsHttp           bool `long:"proposalshttp" description:"Enable/Disable the proposals http module from running"`
	EnableAgendas                 bool `long:"agendas" description:"Enable/Disable the agendas module from running"`
	EnableAgendasHttp             bool `long:"agendashttp" description:"Enable/Disable the agendas http module from running"`
	EnableExchange                bool `long:"exchange" description:"Enable/Disable the exchange historic data collector from running"`
	EnableExchangeHttp            bool `long:"exchange-http" description:"Enable/Disable the exchange historic http endpoint from running"`
	EnablePow                     bool `long:"pow" description:"Enable/Disable PoW module from running"`
	EnablePowHttp                 bool `long:"powhttp" description:"Enable/Disable PoW http endpoint from running"`
	EnableVSP                     bool `long:"vsp" description:"Enable/Disable VSP module from running"`
	EnableVSPHttp                 bool `long:"vsphttp" description:"Enable/Disable VSP http endpoint from running"`
	EnableStats                   bool `long:"stats" description:"Enable/Disable Stats endpoint from running"`
	EnableCharts                  bool `long:"charts" description:"Enable/Disable Charts"`
	EnableTreasuryChart           bool `long:"treasury-chart" description:"Enable/Disable treasury chart module"`

	// Mempool
	MempoolInterval float64 `long:"mempoolinterval" description:"The duration of time between mempool collection"`

	// Propagation
	PropDBHost []string `long:"propdbhost" description:"Propagation database host"`
	PropDBPort []string `long:"propdbport" description:"Propagation database port"`
	PropDBUser []string `long:"propdbuser" description:"Propagation database username"`
	PropDBPass []string `long:"propdbpass" description:"Propagation database password"`
	PropDBName []string `long:"propdbname" description:"Database with external block propagation entry for comparison. Must comatain block and vote tables"`

	// pow
	DisabledPows []string `long:"disabledpow" description:"Disable data collection for this Pow"`
	PowInterval  int64    `long:"powinterval" description:"Collection interval for Pow"`

	// vsp
	VSPInterval int64 `long:"vspinterval" description:"Collection interval for pool status collection"`

	netsnapshot.NetworkSnapshotOptions
	commstats.CommunityStatOptions
}

func defaultConfig() config {
	cfg := config{
		HomeDir:                       defaultHomeDir,
		DataDir:                       defaultDataDir,
		LogDir:                        defaultLogDir,
		DBHost:                        defaultDbHost,
		DBPort:                        defaultDbPort,
		DBUser:                        defaultDbUser,
		DBPass:                        defaultDbPass,
		DBName:                        defaultDbName,
		MaxLogZips:                    defaultMaxLogZips,
		ConfigFile:                    defaultConfigFile,
		DebugLevel:                    defaultLogLevel,
		AgendasDBFileName:             defaultAgendasDBFileName,
		ProposalsFileName:             defaultProposalsFileName,
		PoliteiaAPIURL:                defaultPoliteiaAPIURl,
		PiPropRepoOwner:               defaultPiPropoalRepoOwner,
		PiPropRepoName:                defaultPiProposalRepo,
		HTTPProfPath:                  defaultHTTPProfPath,
		APIProto:                      defaultAPIProto,
		APIURL:                        defaultAPIURL,
		CacheControlMaxAge:            defaultCacheControlMaxAge,
		ServerHeader:                  defaultServerHeader,
		DcrdCert:                      defaultDaemonRPCCertFile,
		ExchangeCurrency:              defaultExchangeIndex,
		DisabledExchanges:             defaultDisabledExchanges,
		RateCertificate:               defaultRateCertFile,
		MainnetLink:                   defaultMainnetLink,
		TestnetLink:                   defaultTestnetLink,
		OnionAddress:                  defaultOnionAddress,
		EnableStakingRewardCalculator: true,
		EnableChainParameters:         true,
		EnableAttackCost:              true,
		EnableMempool:                 true,
		EnablePropagation:             true,
		EnableProposals:               true,
		EnableProposalsHttp:           true,
		EnableAgendas:                 true,
		EnableAgendasHttp:             true,
		EnableExchange:                true,
		EnableExchangeHttp:            true,
		EnablePow:                     true,
		EnablePowHttp:                 true,
		EnableVSP:                     true,
		EnableVSPHttp:                 true,
		EnableStats:                   true,
		EnableCharts:                  true,
		EnableTreasuryChart:           true,

		MempoolInterval: defaultMempoolInterval,
		PowInterval:     int64(defaultPowInterval),
		VSPInterval:     int64(defaultVSPInterval),
	}
	cfg.EnableNetworkSnapshot = true
	cfg.EnableNetworkSnapshotHTTP = true
	cfg.SnapshotInterval = defaultSnapshotInterval
	cfg.Seeder = defaultSeeder
	cfg.SeederPort = uint16(defaultSeederPort)
	cfg.MaxPeerConnectionFailure = maxPeerConnectionFailure

	cfg.CommunityStat = true
	cfg.CommunityStatHttp = true
	cfg.RedditStatInterval = defaultRedditInterval
	cfg.Subreddit = defaultSubreddits
	cfg.TwitterStatInterval = defaultTwitterStatInterval
	cfg.TwitterHandles = defaultTwitterHandles
	cfg.GithubStatInterval = defaultGithubStatInterval
	cfg.GithubRepositories = defaultGithubRepositories
	cfg.YoutubeStatInterval = defaultYoutubeInterval
	cfg.YoutubeChannelName = defaultYoutubeChannelNames
	cfg.YoutubeChannelId = defaultYoutubeChannelId

	return cfg
}

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

func copyFile(sourec, destination string) error {
	from, err := os.Open(sourec)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
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
	cfg := defaultConfig()
	defaultConfigNow := defaultConfig()

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

	// if the config file is missing, create the default
	pathNotExists := func(path string) bool {
		_, err := os.Stat(path)
		return os.IsNotExist(err)
	}

	if pathNotExists(preCfg.ConfigFile) {
		if pathNotExists(defaultHomeDir) {
			if err = os.MkdirAll(defaultHomeDir, os.ModePerm); err != nil {
				return nil, fmt.Errorf("Missing config file and cannot create home dir - %s", err.Error())
			}
		}

		if err = copyFile(sampleConfigFileName, preCfg.ConfigFile); err != nil {
			return nil, fmt.Errorf("Missing config file and cannot copy the sample - %s", err.Error())
		}
	}

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
	cfg.DcrdRPCServer, err = normalizeNetworkAddress(cfg.DcrdRPCServer, defaultHost, activeNet.JSONRPCClientPort)
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
	cfg.RateCertificate = cleanAndExpandPath(cfg.RateCertificate)
	cfg.AgendasDBFileName = cleanAndExpandPath(cfg.AgendasDBFileName)
	cfg.ProposalsFileName = cleanAndExpandPath(cfg.ProposalsFileName)

	// Clean up the provided mainnet and testnet links, ensuring there is a single
	// trailing slash.
	cfg.MainnetLink = strings.TrimSuffix(cfg.MainnetLink, "/") + "/"
	cfg.TestnetLink = strings.TrimSuffix(cfg.TestnetLink, "/") + "/"

	return &cfg, nil
}
