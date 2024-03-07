package matching

import (
	"github.com/CheetahExchange/CheetahExchange/models"
)

// Used for matching engine to read order, need to support setting offset, from the specified offset to start reading
type OrderReader interface {
	// Set the start offset for reading
	SetOffset(offset int64) error

	// Pull order
	FetchOrder() (offset int64, order *models.Order, err error)
}

// Used to keep matching logs
type LogStore interface {
	// Save Log
	Store(logs []interface{}) error
}

// Reading matching logs in observer mode
type LogReader interface {
	// Get the current productId
	GetProductId() string

	// Registering a log watcher
	RegisterObserver(observer LogObserver)

	// Start the execution of reading the log, the read log will be called back to the observer
	Run(seq, offset int64)
}

// Matching log reader observer
type LogObserver interface {
	// Callback when OpenLog is read.
	OnOpenLog(log *OpenLog, offset int64)

	// Callback when MatchLog is read
	OnMatchLog(log *MatchLog, offset int64)

	// Callback when DoneLog is read
	OnDoneLog(log *DoneLog, offset int64)
}

// Used to save a snapshot of the matching engine
type SnapshotStore interface {
	// Save Snapshot
	Store(snapshot *Snapshot) error

	// Getting the last snapshot
	GetLatest() (*Snapshot, error)
}
