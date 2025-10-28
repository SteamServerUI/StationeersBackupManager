package backupmgr

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/SteamServerUI/PluginLib"
	"github.com/fsnotify/fsnotify"
)

/*
The BackupManager manages backup operations. Each instance is independent with its own config and context.
Background routines (file watching and cleanup) only start when Start() is called. Multiple instances
can coexist but may conflict if configured with overlapping directories.
*/

// Initialize checks for BackupDir and waits until it exists, then ensures SafeBackupDir exists.
// It returns a channel that signals when initialization is complete or an error occurs.
func (m *BackupManager) Initialize(identifier string) <-chan error {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(chan error, 1)

	go func() {
		defer close(result)
		const timeout = 90 * time.Minute
		const pollInterval = 2500 * time.Millisecond
		deadline := time.Now().Add(timeout)

		// Wait for BackupDir to exist
		for time.Now().Before(deadline) {
			if stat, err := os.Stat(m.config.BackupDir); err == nil {
				if stat.IsDir() {
					// Directory exists, proceed
					PluginLib.Log(fmt.Sprintf("%s found backup directory: %s", identifier, m.config.BackupDir), "Debug")
					break
				}
				result <- fmt.Errorf("%s backup path %s is not a directory", identifier, m.config.BackupDir)
				return
			} else if !os.IsNotExist(err) {
				// An error other than "not exists" occurred
				result <- fmt.Errorf("%s error checking backup directory %s: %v", identifier, m.config.BackupDir, err)
				return
			}

			err := PluginLib.Log(fmt.Sprintf("%s waiting for save folder %s to be created by Stationeers...", identifier, m.config.BackupDir), "Debug")
			if err != nil {
				fmt.Println(identifier)
				fmt.Println(err.Error())
			}
			select {
			case <-m.ctx.Done():
				result <- fmt.Errorf("%s I have to go, the config was likely changed: %s", identifier, m.ctx.Err())
				return
			case <-time.After(pollInterval):
				// Continue polling
			}
		}

		if time.Now().After(deadline) {
			result <- fmt.Errorf("%s timeout waiting for backup directory %s to be created", identifier, m.config.BackupDir)
			return
		}

		// Ensure SafeBackupDir exists, create it if it doesn't
		if err := os.MkdirAll(m.config.SafeBackupDir, os.ModePerm); err != nil {
			result <- fmt.Errorf("%s error creating safe backup directory %s: %v", identifier, m.config.SafeBackupDir, err)
			return
		}
		PluginLib.Log(fmt.Sprintf("%s created safebackups at %s", identifier, m.config.SafeBackupDir), "Debug")

		result <- nil
	}()

	return result
}

// Start begins the backup monitoring and cleanup routines
func (m *BackupManager) Start(identifier string) error {
	// Wait for initialization to complete
	PluginLib.Log(fmt.Sprintf("%s is waiting for save folder initialization...", identifier), "Debug")
	initResult := <-m.Initialize(identifier)
	if initResult != nil {
		return fmt.Errorf("%s failed to initialize backup manager : %w", identifier, initResult)
	}
	PluginLib.Log(fmt.Sprintf("%s Backup manager instance started", identifier), "Info")

	// Start file watcher
	watcher, err := newFsWatcher(m.config.BackupDir, identifier)
	if err != nil {
		return fmt.Errorf("failed to create autosave watcher: %w", err)
	}
	m.watcher = watcher
	go m.watchBackups(identifier)

	return nil
}

// watchBackups monitors the backup directory for new files
func (m *BackupManager) watchBackups(identifier string) {
	m.wg.Add(1)
	defer m.wg.Done()

	PluginLib.Log(fmt.Sprintf("%s Starting backup file watcher...", identifier), "Debug")
	defer PluginLib.Log(fmt.Sprintf("%s Backup file watcher stopped", identifier), "Info")

	for {
		select {
		case <-m.ctx.Done():
			PluginLib.Log(fmt.Sprintf("%s WatchBackups stopped due to context cancellation", identifier), "Info")
			return
		case event, ok := <-m.watcher.events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				PluginLib.Log(fmt.Sprintf("%s New backup file detected: %s", identifier, event.Name), "Info")
				m.handleNewBackup(event.Name)
			}
		case err, ok := <-m.watcher.errors:
			if !ok {
				return
			}
			PluginLib.Log(fmt.Sprintf("%s Backup watcher error: %s", identifier, err.Error()), "Error")
		}
	}
}

// handleNewBackup processes a newly created backup file
func (m *BackupManager) handleNewBackup(filePath string) {
	if !isValidBackupFile(filepath.Base(filePath)) {
		return
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		time.Sleep(m.config.WaitTime)

		m.mu.Lock()
		defer m.mu.Unlock()

		fileName := filepath.Base(filePath)
		relativePath, err := filepath.Rel(m.config.BackupDir, filePath)
		if err != nil {
			PluginLib.Log(fmt.Sprintf("Error getting relative path for %s: %s", filePath, err.Error()), "Error")
			return
		}
		dstPath := filepath.Join(m.config.SafeBackupDir, relativePath)

		if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
			PluginLib.Log(fmt.Sprintf("Error creating destination dir for %s: %s", dstPath, err.Error()), "Error")
			return
		}

		if err := copyFile(filePath, dstPath); err != nil {
			PluginLib.Log(fmt.Sprintf("Error copying backup %s: %s", fileName, err.Error()), "Error")
			return
		}

		PluginLib.Log(fmt.Sprintf("Backup successfully copied to safe location: %s", dstPath), "Info")
	}()
}

// ListBackups returns information about available backups
// limit: number of recent backups to return (0 for all)
func (m *BackupManager) ListBackups(limit int) ([]BackupGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	groups, err := m.getBackupGroups()
	if err != nil {
		return nil, err
	}

	// Sort by index (newest first)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Index > groups[j].Index
	})

	if limit > 0 && limit < len(groups) {
		groups = groups[:limit]
	}

	return groups, nil
}

// Shutdown stops all backup operations
func (m *BackupManager) Shutdown() {
	PluginLib.Log("Shutting down previous backup manager...", "Info")

	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
		PluginLib.Log("Context canceled for previous backup manager", "Info")
	}

	if m.watcher != nil {
		m.watcher.close()
		m.watcher = nil
		PluginLib.Log("File watcher closed", "Info")
	}
	m.mu.Unlock()

	// Wait for all goroutines to finish
	PluginLib.Log("Waiting for background tasks to complete...", "Info")
	m.wg.Wait()

	PluginLib.Log("Backup manager shut down completely", "Info")
}

// NewBackupManager creates a new BackupManager instance
func NewBackupManager(cfg BackupConfig) *BackupManager {
	ctx, cancel := context.WithCancel(context.Background())

	if cfg.WaitTime == 0 {
		cfg.WaitTime = defaultWaitTime
	}

	return &BackupManager{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}
