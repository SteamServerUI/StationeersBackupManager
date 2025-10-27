package main

import (
	"embed"
	"fmt"
	"log"
	"sync"

	"github.com/SteamServerUI/PluginLib"
	"github.com/SteamServerUI/StationeersBackupManager/api"
	"github.com/SteamServerUI/StationeersBackupManager/backupmgr"
	"github.com/SteamServerUI/StationeersBackupManager/global"
)

//go:embed assets/*
var assets embed.FS

var (
	settingsResponse PluginLib.SettingsResponse
	wg               sync.WaitGroup
)

func main() {

	// Register embedded assets
	global.AssetManager = PluginLib.RegisterAssets(&assets)

	PluginLib.InitConfig(global.PluginName, global.DefaultLogLevel)

	if err := backupmgr.ReloadBackupManagerFromConfig(); err != nil {
		PluginLib.Log("Failed to reload backup manager: " + err.Error())
		return
	}

	ExposeAPI(&wg)
	wg.Wait()
}

func GetGameserverRunningStatus() {
	rsp, err := PluginLib.GetServerStatus()
	if err != nil {
		log.Fatalf("Failed to get server status: %v", err)
	}

	fmt.Println("Gameserver running:", rsp.Status, "UUID:", rsp.UUID)
}

func LogSomething() {
	// allows either a message
	PluginLib.Log("Test")
	// or a message and a log level
	PluginLib.Log("Test", "Info")
	// also allows proper error handling
	err := PluginLib.Log("Test", "Non-Existing-Level")
	if err != nil {
		fmt.Println("Error (expected, since level doesn't exist):", err)
	}
}

func SaveASetting() {

	payload := map[string]string{"GameBranch": "public"}

	if _, err := PluginLib.Post("/api/v2/settings/save", &payload, &settingsResponse); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Setting saved:", settingsResponse.Message)
}

func GetSetting() {
	value, err := PluginLib.GetSetting("GameBranch")
	if err != nil {
		log.Printf("Failed to get setting: %v", err)
		return
	}
	fmt.Printf("Setting 'GameBranch': %v\n", value)
}

func GetAllSettings() {
	settings := PluginLib.GetAllSettings()
	for key, value := range settings {
		fmt.Printf("Setting '%s': %v\n", key, value)
	}
}

func ExposeAPI(wg *sync.WaitGroup) {

	rfi, err := getRfIdentifierFromSSUIRunfile()
	if err != nil {
		log.Fatal(err)
	}
	if rfi == "" {
		log.Fatal("RunfileIdentifier is empty")
	}

	if rfi != "Stationeers" && rfi != "StationeersNewTerrain" {
		log.Fatal("RunfileIdentifier is not Stationeers or StationeersNewTerrain")
	}

	global.RunfileIdentifier = rfi

	backupHandler := backupmgr.NewHTTPHandler(backupmgr.GlobalBackupManager)
	PluginLib.RegisterRoute("/", api.HandleBackupManagerIndex)
	PluginLib.RegisterRoute("/js/backups.js", api.HandleBackupsJS)

	PluginLib.RegisterRoute("/api/v1/backups", backupHandler.ListBackupsHandler)
	PluginLib.RegisterRoute("/api/v1/backups/restore", backupHandler.RestoreBackupHandler)
	PluginLib.ExposeAPI(wg)
	PluginLib.RegisterPluginAPI()
	wg.Add(1)
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
