package netsnapshot

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/decred/dcrd/wire"
	"github.com/planetdecred/pdanalytics/web"
)

type SnapShot struct {
	Timestamp           int64  `json:"timestamp"`
	Height              int64  `json:"height"`
	NodeCount           int    `json:"node_count"`
	ReachableNodeCount  int    `json:"reachable_node_count"`
	OldestNode          string `json:"oldest_node"`
	OldestNodeTimestamp int64  `json:"oldest_node_timestamp"`
	Latency             int    `json:"latency"`
}

type NodeCount struct {
	Timestamp int64 `json:"timestamp"`
	Count     int64 `json:"count"`
}

type UserAgentInfo struct {
	UserAgent string `json:"user_agent"`
	Nodes     int64  `json:"nodes"`
	Timestamp int64  `json:"timestamp"`
	Height    int64  `json:"height"`
}

type CountryInfo struct {
	Country   string `json:"country"`
	Nodes     int64  `json:"nodes"`
	Timestamp int64  `json:"timestamp"`
	Height    int64  `json:"height"`
}

type NetworkPeer struct {
	Timestamp       int64  `json:"timestamp"`
	Address         string `json:"address"`
	UserAgent       string `json:"user_agent"`
	StartingHeight  int64  `json:"starting_height"`
	CurrentHeight   int64  `json:"current_height"`
	ConnectionTime  int64  `json:"connection_time"`
	ProtocolVersion uint32 `json:"protocol_version"`
	LastSeen        int64  `json:"last_seen"`
	LastSuccess     int64  `json:"last_success"`
	IsDead          bool   `json:"is_dead"`
	Latency         int    `json:"latency"`
	Reachable       bool   `json:"reachable"`
	IPVersion       int    `json:"ip_version"`
	Services        string `json:"services"`
	LastAttempt     int64  `json:"last_attempt"`

	IPInfo
}

type Heartbeat struct {
	Timestamp     int64  `json:"timestamp"`
	Address       string `json:"address"`
	LastSeen      int64  `json:"last_seen"`
	Latency       int    `json:"latency"`
	CurrentHeight int64  `json:"current_height"`
}

type IPInfo struct {
	Type        string `json:"type"`
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	RegionCode  string `json:"region_code"`
	RegionName  string `json:"region_name"`
	City        string `json:"city"`
	Zip         string `json:"zip"`
}

type DataStore interface {
	LastSnapshotTime(ctx context.Context) (timestamp int64)
	DeleteSnapshot(ctx context.Context, timestamp int64)
	SaveSnapshot(ctx context.Context, snapShot SnapShot) error
	UpdateSnapshotNodesBin(ctx context.Context) error
	SaveHeartbeat(ctx context.Context, peer Heartbeat) error
	AttemptPeer(ctx context.Context, address string, now int64) error
	RecordNodeConnectionFailure(ctx context.Context, address string, maxAllowedFailure int) error
	SaveNode(ctx context.Context, peer NetworkPeer) error
	UpdateNode(ctx context.Context, peer NetworkPeer) error
	GetAvailableNodes(ctx context.Context) ([]net.IP, error)
	LastSnapshot(ctx context.Context) (*SnapShot, error)
	GetIPLocation(ctx context.Context, ip string) (string, int, error)
	FindNode(ctx context.Context, address string) (*NetworkPeer, error)
	NodeExists(ctx context.Context, address string) (bool, error)

	Snapshots(ctx context.Context, offset, limit int, forChart bool) ([]SnapShot, int64, error)
	SnapshotCount(ctx context.Context) (int64, error)
	FindNetworkSnapshot(ctx context.Context, timestamp int64) (*SnapShot, error)
	PreviousSnapshot(ctx context.Context, timestamp int64) (*SnapShot, error)
	NextSnapshot(ctx context.Context, timestamp int64) (*SnapShot, error)
	TotalPeerCount(ctx context.Context, timestamp int64) (int64, error)
	SeenNodesByTimestamp(ctx context.Context) ([]NodeCount, error)
	NetworkPeers(ctx context.Context, timestamp int64, q string, offset int, limit int) ([]NetworkPeer, int64, error)
	NetworkPeer(ctx context.Context, address string) (*NetworkPeer, error)
	AverageLatency(ctx context.Context, address string) (int, error)
	PeerCountByUserAgents(ctx context.Context, sources string, offset, limit int) (userAgents []UserAgentInfo, total int64, err error)
	PeerCountByIPVersion(ctx context.Context, timestamp int64, iPVersion int) (int64, error)
	PeerCountByCountries(ctx context.Context, sources string, offset, limit int) (countries []CountryInfo, total int64, err error)
	AllNodeVersions(ctx context.Context) ([]string, error)
	AllNodeContries(ctx context.Context) ([]string, error)
	FetchNodeLocations(ctx context.Context, offset, limit int) ([]CountryInfo, int64, error)
	FetchNodeVersion(ctx context.Context, offset, limit int) ([]UserAgentInfo, int64, error)
	SnapshotsByTime(ctx context.Context, startDate int64, pageSize int) ([]SnapShot, error)
	SnapshotsByBin(ctx context.Context, bin string) ([]SnapShot, error)
	NodeVersionsByBin(ctx context.Context, userAgent, bin string) ([]UserAgentInfo, error)
	NodeLocationsByBin(ctx context.Context, userAgent, bin string) ([]CountryInfo, error)
}

type Node struct {
	IP           net.IP
	Port         uint16
	Services     wire.ServiceFlag
	LastAttempt  time.Time
	AttemptCount int
	LastSuccess  time.Time
	LastSeen     time.Time
	Latency      int64

	ConnectionTime  int64
	ProtocolVersion uint32
	UserAgent       string
	StartingHeight  int64
	CurrentHeight   int64
	IPVersion       int
}

type peerAddress struct {
	IP   net.IP
	Port uint16
}

type attemptedPeer struct {
	IP   net.IP
	Time int64
}

type Manager struct {
	mtx sync.RWMutex

	nodes        map[string]*Node
	liveNodeIPs  []net.IP
	peerNtfn     chan *Node
	attemptNtfn  chan attemptedPeer
	connFailNtfn chan net.IP
	quit         chan struct{}
	peersFile    string
}

type NetworkSnapshotOptions struct {
	TestNet                   bool   `long:"snapshot-testnet" description:"Use testnet"`
	EnableNetworkSnapshot     bool   `long:"snapshot" description:"Enable/Disable network snapshot taker from running"`
	EnableNetworkSnapshotHTTP bool   `long:"snapshot-http" description:"Enable/Disable network snapshot web request handler from running"`
	SeederPort                uint16 `long:"seederport" description:"Port of a working node"`
	Seeder                    string `short:"s" long:"seeder" description:"IP address of a working node"`
	IpStackAccessKey          string `long:"ip-stack-access-key" description:"IP stack access key https://ipstack.com/"`
	IpLocationProvidingPeer   string `long:"ip-location-providing-peer" description:"An optional peer address for getting IP info"`
	SnapshotInterval          int    `long:"snapshotinterval" description:"The number of minutes between snapshot (default 5)"`
	MaxPeerConnectionFailure  int    `long:"max-peer-connection-failure" description:"Number of failed connection before a pair is marked a dead"`
}

type taker struct {
	dataStore DataStore
	cfg       NetworkSnapshotOptions
	server    *web.Server
}
