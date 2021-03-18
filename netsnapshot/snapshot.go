// netsnapshot contain logics that craws the dcrd network to record the heart beat of
// active node. The information is presented and tabular/chart view on the web.
// The snapshot taker and web request handlers can be toggled on/off from the config option

package netsnapshot

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/decred/dcrd/chaincfg/v2"
	"github.com/planetdecred/pdanalytics/web"
)

var snapshotinterval int

func Snapshotinterval() int {
	return snapshotinterval
}

func Activate(ctx context.Context, store DataStore, cfg NetworkSnapshotOptions, server *web.Server) error {
	snapshotinterval = cfg.SnapshotInterval
	t := &taker{
		dataStore: store,
		server:    server,
		cfg:       cfg,
	}

	if cfg.EnableNetworkSnapshot {
		go t.Start(ctx)
	}

	if cfg.EnableNetworkSnapshotHTTP {
		if err := t.configHTTPHandlers(); err != nil {
			return err
		}
	}

	return nil
}

func (t *taker) Start(ctx context.Context) {
	log.Info("Triggering network snapshot taker.")

	var netParams = chaincfg.MainNetParams()
	if t.cfg.TestNet {
		netParams = chaincfg.TestNet3Params()
	}

	// defaultStaleTimeout = time.Minute * time.Duration(t.cfg.SnapshotInterval)
	// pruneExpireTimeout = defaultStaleTimeout * 2

	var err error
	amgr, err = NewManager(filepath.Join(defaultHomeDir, netParams.Name), t.cfg.SnapshotInterval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewManager: %v\n", err)
		os.Exit(1)
	}

	// update all reachable nodes
	loadLiveNodes := func() {
		nodes, err := t.dataStore.GetAvailableNodes(ctx)
		if err != nil {
			log.Errorf("Error in taking network snapshot, %s", err.Error())
		}
		amgr.setLiveNodes(nodes)
	}

	// enqueue previous known ips
	loadLiveNodes()

	go runSeeder(t.cfg, netParams)

	var mtx sync.Mutex
	var bestBlockHeight int64

	var count int
	var timestamp = time.Now().UTC().Unix()
	snapshot := SnapShot{
		Timestamp: timestamp,
		Height:    bestBlockHeight,
	}

	lastSnapshot, err := t.dataStore.LastSnapshot(ctx)
	if err == nil {
		minutesPassed := math.Abs(time.Since(time.Unix(lastSnapshot.Timestamp, 0)).Minutes())
		if minutesPassed < float64(t.cfg.SnapshotInterval) {
			snapshot = *lastSnapshot
			timestamp = lastSnapshot.Timestamp
		}
	}

	snapshot.NodeCount = len(amgr.nodes)
	if snapshot.NodeCount > 0 {
		if err = t.dataStore.SaveSnapshot(ctx, snapshot); err != nil {
			t.dataStore.DeleteSnapshot(ctx, timestamp)
			log.Errorf("Error in saving network snapshot, %s", err.Error())
		}
	}

	ticker := time.NewTicker(time.Duration(t.cfg.SnapshotInterval) * time.Minute)
	defer ticker.Stop()

	for {
		// start listening for node heartbeat
		select {
		case <-ticker.C:
			err := t.dataStore.SaveSnapshot(ctx, SnapShot{
				Timestamp: timestamp,
				Height:    bestBlockHeight,
				NodeCount: len(amgr.nodes),
			})

			if err != nil {
				t.dataStore.DeleteSnapshot(ctx, timestamp)
				log.Errorf("Error in saving network snapshot, %s", err.Error())
			}
			log.Info("UpdateSnapshotNodesBin")
			if err = t.dataStore.UpdateSnapshotNodesBin(ctx); err != nil {
				log.Errorf("Error in initial network snapshot bin update, %s", err.Error())
			}

			mtx.Lock()
			count = 0
			log.Infof("Took a new network snapshot, recorded %d discoverable nodes.", count)
			timestamp = time.Now().UTC().Unix()
			mtx.Unlock()
			// update all reachable nodes
			loadLiveNodes()

		case node := <-amgr.peerNtfn:
			if node.IP.String() == "127.0.0.1" { // do not add the local IP
				break
			}

			networkPeer := NetworkPeer{
				Timestamp:       timestamp,
				Address:         node.IP.String(),
				LastAttempt:     node.LastAttempt.UTC().Unix(),
				LastSeen:        node.LastSeen.UTC().Unix(),
				LastSuccess:     node.LastSuccess.UTC().Unix(),
				ConnectionTime:  node.ConnectionTime,
				ProtocolVersion: node.ProtocolVersion,
				UserAgent:       node.UserAgent,
				StartingHeight:  node.StartingHeight,
				CurrentHeight:   node.CurrentHeight,
				Services:        node.Services.String(),
				Latency:         int(node.Latency),
			}

			if exists, _ := t.dataStore.NodeExists(ctx, networkPeer.Address); exists {
				err = t.dataStore.UpdateNode(ctx, networkPeer)
				if err != nil {
					log.Errorf("Error in saving node info, %s.", err.Error())
				}
			} else {
				geoLoc, err := t.geolocation(ctx, node.IP)
				if err == nil {
					networkPeer.IPInfo = *geoLoc
					// networkPeer.Country = geoLoc.CountryName
					if geoLoc.Type == "ipv4" {
						networkPeer.IPVersion = 4
					} else if geoLoc.Type == "ipv6" {
						networkPeer.IPVersion = 6
					}
				} else {
					log.Error(err)
				}

				err = t.dataStore.SaveNode(ctx, networkPeer)
				if err != nil {
					log.Errorf("Error in saving node info, %s.", err.Error())
				}
			}

			err = t.dataStore.SaveHeartbeat(ctx, Heartbeat{
				Timestamp: timestamp,
				Address:   node.IP.String(),
				LastSeen:  node.LastSeen.UTC().Unix(),
				Latency:   int(node.Latency),
			})
			if err != nil {
				log.Errorf("Error in saving node info, %s.", err.Error())
			} else {
				mtx.Lock()
				count++
				if node.CurrentHeight > bestBlockHeight {
					bestBlockHeight = node.CurrentHeight
				}

				snapshot := SnapShot{
					Timestamp: timestamp,
					Height:    bestBlockHeight,
				}

				snapshot.NodeCount = len(amgr.nodes)
				err = t.dataStore.SaveSnapshot(ctx, snapshot)
				if err != nil {
					// todo delete all the related node info
					t.dataStore.DeleteSnapshot(ctx, timestamp)
					log.Errorf("Error in saving network snapshot, %s", err.Error())
				}

				mtx.Unlock()
				log.Debugf("New heartbeat recorded for node: %s, %s, %d", node.IP.String(),
					node.UserAgent, node.ProtocolVersion)
			}

		case attemptedPeer := <-amgr.attemptNtfn:
			if err := t.dataStore.AttemptPeer(ctx, attemptedPeer.IP.String(), attemptedPeer.Time); err != nil {
				log.Errorf("Error in saving peer attempt for %s, %s", attemptedPeer.IP.String(), err.Error())
			}

		case ip := <-amgr.connFailNtfn:
			if err := t.dataStore.RecordNodeConnectionFailure(ctx, ip.String(), t.cfg.MaxPeerConnectionFailure); err != nil {
				log.Errorf("Error in failed connection attempt for %s, %s", ip.String(), err.Error())
			}

		case <-ctx.Done():
			log.Info("Shutting down network seeder")
			amgr.quit <- struct{}{}
			return
		}
	}
}

func (t *taker) geolocation(ctx context.Context, ip net.IP) (*IPInfo, error) {
	// IP stack access key verification
	if t.cfg.IpStackAccessKey == "" {
		return nil, errors.New("IP stack access key is required")
	}
	url := fmt.Sprintf("http://api.ipstack.com/%s?access_key=%s&format=1", ip.String(), t.cfg.IpStackAccessKey)
	var geo IPInfo
	err := web.GetResponse(ctx, &http.Client{Timeout: 3 * time.Second}, url, &geo)
	return &geo, err
}

func (t *taker) configHTTPHandlers() error {
	if err := t.server.Templates.AddTemplate("nodes"); err != nil {
		log.Errorf("Unable to create new html template: %v", err)
		return err
	}

	t.server.AddMenuItem(web.MenuItem{
		Href:      "/nodes",
		HyperText: "Nodes",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Network nodes",
		},
	})

	t.server.AddRoute("/nodes", web.GET, t.nodesPage)
	t.server.AddRoute("/api/charts/snapshot/{chartDataType}", web.GET, t.chart, web.ChartDataTypeCtx)
	t.server.AddRoute("/api/snapshots", web.GET, t.snapshots)
	t.server.AddRoute("/api/snapshots/user-agents", web.GET, t.nodesCountUserAgents)
	t.server.AddRoute("/api/snapshots/user-agents/chart", web.GET, t.nodesCountUserAgentsChart)
	t.server.AddRoute("/api/snapshots/countries", web.GET, t.nodesCountByCountries)
	t.server.AddRoute("/api/snapshots/countries/chart", web.GET, t.nodesCountByCountriesChart)
	t.server.AddRoute("/api/snapshot/nodes/count-by-timestamp", web.GET, t.nodeCountByTimestamp)
	t.server.AddRoute("/api/snapshot/node-versions", web.GET, t.nodeVersions)
	t.server.AddRoute("/api/snapshot/node-countries", web.GET, t.nodeCountries)

	return nil
}
