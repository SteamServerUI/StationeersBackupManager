package backupmgr

import (
	"fmt"
	"sync"
	"time"

	"github.com/SteamServerUI/PluginLib"
	"github.com/google/uuid"
)

// GlobalBackupManager is the singleton instance of the backup manager
var GlobalBackupManager *BackupManager

// Track all HTTP handlers that need updating when manager changes
var activeHTTPHandlers []*HTTPHandler

// initMutex ensures thread-safe initialization of the global backup manager
var initMutex sync.Mutex

// InitGlobalBackupManager initializes the global backup manager instance
func InitGlobalBackupManager(config BackupConfig) error {
	// Lock to prevent concurrent initialization
	initMutex.Lock()
	defer initMutex.Unlock()

	// Shut down existing manager if it exists
	if GlobalBackupManager != nil {
		PluginLib.Log(fmt.Sprintf("%s Previous Backup manager found. Shutting it down.", config.Identifier), "Info")
		GlobalBackupManager.Shutdown()
		GlobalBackupManager = nil // Clear the manager to avoid stale references
	}

	PluginLib.Log(fmt.Sprintf("%s Creating a global backup manager with ID %s", config.Identifier, config.Identifier), "Info")
	manager := NewBackupManager(config)
	GlobalBackupManager = manager

	// Update all active HTTP handlers with the new manager
	for _, handler := range activeHTTPHandlers {
		handler.manager = GlobalBackupManager
	}

	// Start the backup manager in a goroutine to avoid blocking
	go func(m *BackupManager) {
		if err := m.Start(config.Identifier); err != nil {
			PluginLib.Log(fmt.Sprintf("%s Exited: %s", config.Identifier, err.Error()), "Error")
		}
	}(manager)

	PluginLib.Log(fmt.Sprintf("%s Backup manager reloaded successfully", config.Identifier), "Info")
	return nil
}

// RegisterHTTPHandler registers an HTTP handler to be updated when the manager changes
func RegisterHTTPHandler(handler *HTTPHandler) {
	activeHTTPHandlers = append(activeHTTPHandlers, handler)
}

// GetBackupConfig returns a properly configured BackupConfig
func GetBackupConfig() BackupConfig {

	id := uuid.New()
	bmIdentifier := "[BM" + id.String()[:6] + "]:"
	return BackupConfig{
		WorldName:     "SaveName",
		BackupDir:     "./saves/SaveName/autosave",
		SafeBackupDir: "./saves/SaveName/Safebackups",
		WaitTime:      30 * time.Second, // not sure why we are not using config.BackupWaitTime here, but ill not touch it in this commit (config rework)
		RetentionPolicy: RetentionPolicy{
			KeepLastN:       0,
			KeepWeeklyFor:   0,
			KeepMonthlyFor:  0,
			CleanupInterval: 0,
		},
		Identifier: bmIdentifier,
	}
}

// ReloadBackupManagerFromConfig reloads the global backup manager with the current config. This should be called whenever the config is changed.
func ReloadBackupManagerFromConfig() error {
	// Create a new backupManager config from the global config
	backupConfig := GetBackupConfig()

	// Reinitialize the global backup manager with the new config
	return InitGlobalBackupManager(backupConfig)
}
