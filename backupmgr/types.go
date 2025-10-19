package backupmgr

import (
	"context"
	"sync"
	"time"
)

const (
	defaultWaitTime = 30 * time.Second
)

// BackupConfig holds configuration for backup operations
type BackupConfig struct {
	WorldName     string
	BackupDir     string
	SafeBackupDir string
	WaitTime      time.Duration
	Identifier    string
}

// BackupGroup represents a set of backup files
type BackupGroup struct {
	Index    int
	BinFile  string
	XMLFile  string
	MetaFile string
	ModTime  time.Time
}

// BackupManager manages backup operations
type BackupManager struct {
	config  BackupConfig
	mu      sync.Mutex
	watcher *fsWatcher
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup // Added for tracking goroutines
}
