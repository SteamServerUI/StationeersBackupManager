package backupmgr

import (
	"fmt"
	"os"
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

	PluginLib.Log(fmt.Sprintf("%s Backup manager reloaded successfully", config.Identifier), "Debug")
	return nil
}

// RegisterHTTPHandler registers an HTTP handler to be updated when the manager changes
func RegisterHTTPHandler(handler *HTTPHandler) {
	activeHTTPHandlers = append(activeHTTPHandlers, handler)
}

// GetBackupConfig returns a properly configured BackupConfig
func GetBackupConfig() BackupConfig {

	saveName, err := getSaveNameFromSSUIRunfile()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	runfileIdentifier, err := getRfIdentifierFromSSUIRunfile()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	id := uuid.New()
	bmIdentifier := "[BM" + id.String()[:6] + "]:"
	return BackupConfig{
		WorldName:     "SaveName",
		BackupDir:     "./" + runfileIdentifier + "/saves/" + saveName + "/autosave",
		SafeBackupDir: "./" + runfileIdentifier + "/saves/" + saveName + "/Safebackups",
		WaitTime:      20 * time.Second,
		Identifier:    bmIdentifier,
	}
}

// ReloadBackupManagerFromConfig reloads the global backup manager with the current config. This should be called whenever the config is changed.
func ReloadBackupManagerFromConfig() error {
	// Create a new backupManager config from the global config
	backupConfig := GetBackupConfig()

	// Reinitialize the global backup manager with the new config
	return InitGlobalBackupManager(backupConfig)
}

func getSaveNameFromSSUIRunfile() (string, error) {
	savename, err := PluginLib.GetSingleArgFromRunfile("SaveName")
	if err != nil {
		return "", fmt.Errorf("failed to get save name from runfile: %w", err)
	}
	return savename, nil
}

func getRfIdentifierFromSSUIRunfile() (string, error) {
	runfileIdentifier, err := PluginLib.GetSetting("RunfileIdentifier")
	if err != nil {
		return "", fmt.Errorf("failed to get RunfileIdentifier from SSUI: %w", err)
	}

	runfileIdentifierStr, ok := runfileIdentifier.(string)
	if !ok {
		return "", fmt.Errorf("RunfileIdentifier is not a string")
	}
	return runfileIdentifierStr, nil
}
