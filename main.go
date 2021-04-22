package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/decred/dcrd/rpcclient/v5"
	"github.com/decred/dcrdata/exchanges/v2"
	"github.com/go-chi/chi"
	"github.com/google/gops/agent"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

func main() {
	// Create a context that is cancelled when a shutdown request is received
	// via requestShutdown.
	ctx := withShutdownCancel(context.Background())
	// Listen for both interrupt signals and shutdown requests.
	go shutdownListener()

	if err := _main(ctx); err != nil {
		if logRotator != nil {
			log.Error(err)
		}
		os.Exit(1)
	}
	os.Exit(0)
}

// _main does all the work. Deferred functions do not run after os.Exit(), so
// main wraps this function, which returns a code.
func _main(ctx context.Context) error {
	// Parse the configuration file, and setup logger.
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load pdanalytics config: %s\n", err.Error())
		return err
	}
	defer func() {
		if logRotator != nil {
			logRotator.Close()
		}
	}()

	if cfg.CPUProfile != "" {
		var f *os.File
		f, err = os.Create(cfg.CPUProfile)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if cfg.UseGops {
		// Start gops diagnostic agent, with shutdown cleanup.
		if err = agent.Listen(agent.Options{}); err != nil {
			return err
		}
		defer agent.Close()
	}

	// Grab a Notifier. After all databases are synced, register handlers with
	// the Register*Group methods, set the best block height with
	// SetPreviousBlock and start receiving notifications with Listen. Create
	// the notifier now so the *rpcclient.NotificationHandlers can be obtained,
	// using (*Notifier).DcrdHandlers, for the rpcclient.Client constructor.
	notifier := dcrd.NewNotifier(ctx)

	// Connect to dcrd RPC server using a websocket.
	dcrdClient, err := connectNodeRPC(cfg, notifier.DcrdHandlers())
	if err != nil || dcrdClient == nil {
		return fmt.Errorf("Connection to dcrd failed: %v", err)
	}

	defer func() {
		if dcrdClient != nil {
			log.Infof("Closing connection to dcrd.")
			dcrdClient.Shutdown()
			dcrdClient.WaitForShutdown()
		}
		log.Infof("Bye!")
		time.Sleep(250 * time.Millisecond)
	}()

	// Display connected network (e.g. mainnet, testnet, simnet).
	curnet, err := dcrdClient.GetCurrentNet()
	if err != nil {
		return fmt.Errorf("Unable to get current network from dcrd: %v", err)
	}
	log.Infof("Connected to dcrd (JSON-RPC API) on %v", curnet.String())

	if curnet != activeNet.Net {
		log.Criticalf("Network of connected node, %s, does not match expected "+
			"network, %s.", activeNet.Net, curnet)
		return fmt.Errorf("expected network %s, got %s", activeNet.Net, curnet)
	}

	var wg sync.WaitGroup

	// ExchangeBot
	var xcBot *exchanges.ExchangeBot
	if cfg.EnableExchangeBot && activeChain.Name != "mainnet" {
		log.Warnf("disabling exchange monitoring. only available on mainnet")
		cfg.EnableExchangeBot = false
	}
	if cfg.EnableExchangeBot {
		botCfg := exchanges.ExchangeBotConfig{
			BtcIndex:       cfg.ExchangeCurrency,
			MasterBot:      cfg.RateMaster,
			MasterCertFile: cfg.RateCertificate,
		}
		if cfg.DisabledExchanges != "" {
			botCfg.Disabled = strings.Split(cfg.DisabledExchanges, ",")
		}
		xcBot, err = exchanges.NewExchangeBot(&botCfg)
		if err != nil {
			log.Errorf("Could not create exchange monitor. Exchange info will be disabled: %v", err)
		} else {
			var xcList, prepend string
			for k := range xcBot.Exchanges {
				xcList += prepend + k
				prepend = ", "
			}
			log.Infof("ExchangeBot monitoring %s", xcList)
			wg.Add(1)
			go xcBot.Start(ctx, &wg)
		}
	}

	webMux := chi.NewRouter()
	webServer, err := web.NewServer(web.Config{
		CacheControlMaxAge: int64(cfg.CacheControlMaxAge),
		Viewsfolder:        "./views",
		AssetsFolder:       "./web/public",
		ReloadHTML:         cfg.ReloadHTML,
	}, webMux, activeChain)
	if err != nil {
		log.Error(err)
		return fmt.Errorf("failed to create new web server (templates missing?)")
	}

	webServer.MountAssetPaths("/", "./web/public")

	err = setupModules(ctx, cfg, &dcrd.Dcrd{
		Rpc:    dcrdClient,
		Params: activeChain,
		Notif:  notifier,
	}, webServer, xcBot)

	if err != nil {
		return err
	}

	// (*notify.Notifier).processBlock will discard incoming block if PrevHash does not match
	bestBlockHash, bestBlockHeight, err := dcrdClient.GetBestBlock()
	if err != nil {
		log.Error(err)
		return fmt.Errorf("Failed to get best block")
	}
	notifier.SetPreviousBlock(*bestBlockHash, uint32(bestBlockHeight))

	// buildExplorer start the web server when its convenient for we are
	// starting here if the block explorer is disable.
	// The action here assumes that all other modules has being configured
	webServer.BuildRoute()
	if !cfg.NoHttp {
		listenAndServeProto(ctx, &wg, cfg.APIListen, cfg.APIProto, webMux)
	}

	// Register for notifications from dcrd. This also sets the daemon RPC
	// client used by other functions in the notify/notification package (i.e.
	// common ancestor identification in processReorg).
	cerr := notifier.Listen(dcrdClient)
	if cerr != nil {
		return fmt.Errorf("RPC client error: %v (%v)", cerr.Error(), cerr.Cause())
	}

	wg.Wait()

	return nil
}

func connectNodeRPC(cfg *config, ntfnHandlers *rpcclient.NotificationHandlers) (*rpcclient.Client, error) {
	var dcrdCerts []byte
	var err error
	if !cfg.DisableDaemonTLS {
		dcrdCerts, err = ioutil.ReadFile(cfg.DcrdCert)
		if err != nil {
			log.Errorf("Failed to read dcrd cert file at %s: %s\n",
				cfg.DcrdCert, err.Error())
			return nil, err
		}
		log.Debugf("Attempting to connect to dcrd RPC %s as user %s "+
			"using certificate located in %s",
			cfg.DcrdRPCServer, cfg.DcrdRPCUser, cfg.DcrdCert)
	} else {
		log.Debugf("Attempting to connect to dcrd RPC %s as user %s (no TLS)",
			cfg.DcrdRPCServer, cfg.DcrdRPCUser)
	}

	connCfg := &rpcclient.ConnConfig{
		Host:         cfg.DcrdRPCServer,
		Endpoint:     "ws",
		User:         cfg.DcrdRPCUser,
		Pass:         cfg.DcrdRPCPassword,
		Certificates: dcrdCerts,
		DisableTLS:   cfg.DisableDaemonTLS,
	}
	return rpcclient.New(connCfg, ntfnHandlers)
}

func listenAndServeProto(ctx context.Context, wg *sync.WaitGroup, listen, proto string, mux http.Handler) {
	// Try to bind web server
	server := http.Server{
		Addr:         listen,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,  // slow requests should not hold connections opened
		WriteTimeout: 60 * time.Second, // hung responses must die
	}

	// Add the graceful shutdown to the waitgroup.
	wg.Add(1)
	go func() {
		// Start graceful shutdown of web server on shutdown signal.
		<-ctx.Done()

		// We received an interrupt signal, shut down.
		log.Infof("Gracefully shutting down web server...")
		if err := server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners.
			log.Infof("HTTP server Shutdown: %v", err)
		}

		// wg.Wait can proceed.
		wg.Done()
	}()

	log.Infof("Now serving the explorer and APIs on %s://%v/", proto, listen)
	// Start the server.
	go func() {
		var err error
		if proto == "https" {
			err = server.ListenAndServeTLS("pdanalytics.cert", "pdanalytics.key")
		} else {
			err = server.ListenAndServe()
		}
		// If the server dies for any reason other than ErrServerClosed (from
		// graceful server.Shutdown), log the error and request pdanalytics be
		// shutdown.
		if err != nil && err != http.ErrServerClosed {
			log.Errorf("Failed to start server: %v", err)
			requestShutdown()
		}
	}()

	// If the server successfully binds to a listening port, ListenAndServe*
	// will block until the server is shutdown. Wait here briefly so the startup
	// operations in main can have a chance to bail out.
	time.Sleep(250 * time.Millisecond)
}

// shutdownRequested checks if the Done channel of the given context has been
// closed. This could indicate cancellation, expiration, or deadline expiry. But
// when called for the context provided by withShutdownCancel, it indicates if
// shutdown has been requested (i.e. via requestShutdown).
func shutdownRequested(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// shutdownRequest is used to initiate shutdown from one of the
// subsystems using the same code paths as when an interrupt signal is received.
var shutdownRequest = make(chan struct{})

// shutdownSignal is closed whenever shutdown is invoked through an interrupt
// signal or from an JSON-RPC stop request.  Any contexts created using
// withShutdownChannel are cancelled when this is closed.
var shutdownSignal = make(chan struct{})

// signals defines the signals that are handled to do a clean shutdown.
// Conditional compilation is used to also include SIGTERM on Unix.
var signals = []os.Signal{os.Interrupt}

// withShutdownCancel creates a copy of a context that is cancelled whenever
// shutdown is invoked through an interrupt signal or from an JSON-RPC stop
// request.
func withShutdownCancel(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-shutdownSignal
		cancel()
	}()
	return ctx
}

// requestShutdown signals for starting the clean shutdown of the process
// through an internal component (such as through the JSON-RPC stop request).
func requestShutdown() {
	shutdownRequest <- struct{}{}
}

// shutdownListener listens for shutdown requests and cancels all contexts
// created from withShutdownCancel.  This function never returns and is intended
// to be spawned in a new goroutine.
func shutdownListener() {
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, signals...)

	// Listen for the initial shutdown signal
	select {
	case sig := <-interruptChannel:
		log.Infof("Received signal (%s). Shutting down...", sig)
	case <-shutdownRequest:
		log.Info("Shutdown requested. Shutting down...")
	}

	// Cancel all contexts created from withShutdownCancel.
	close(shutdownSignal)

	// Listen for any more shutdown signals and log that shutdown has already
	// been signaled.
	for {
		select {
		case <-interruptChannel:
		case <-shutdownRequest:
		}
		log.Info("Shutdown signaled. Already shutting down...")
	}
}
