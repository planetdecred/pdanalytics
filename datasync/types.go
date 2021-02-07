package datasync

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSyncDisabled = errors.New("data sharing is disabled on this instance")
)

type SyncCoordinator struct {
	period      int
	syncers     map[string]Syncer
	syncersKeys map[int]string
	instances   []instance
	isEnabled   bool
}

type instance struct {
	database string
	store    Store
	url      string
}

type Syncer struct {
	LastEntry func(ctx context.Context, db Store) (string, error)
	Collect   func(ctx context.Context, url string) (*Result, error)
	Retrieve  func(ctx context.Context, last string, skip, take int) (*Result, error)
	Append    func(ctx context.Context, db Store, data interface{})
}

type Store interface {
	TableNames() []string
	LastEntry(ctx context.Context, tableName string, receiver interface{}) error

	SaveFromSync(ctx context.Context, item interface{}) error
	ProcessEntries(ctx context.Context) error
}

type Request struct {
	Table        string
	Date         time.Time
	MaxSkipCount int
	MaxTakeCount int
}

type Result struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message,omitempty"`
	Records    interface{} `json:"records,omitempty"`
	TotalCount int64       `json:"total_count,omitempty"`
}
