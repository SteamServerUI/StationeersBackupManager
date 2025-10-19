package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/SteamServerUI/StationeersBackupManagerPlugin/global"
)

func HandleBackupManagerIndex(w http.ResponseWriter, r *http.Request) {
	data, err := global.AssetManager.GetAssetString("assets/index.html")
	if err != nil {
		log.Fatalf("Failed to read asset: %v", err)
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "%s", data)
}

func HandleBackupsJS(w http.ResponseWriter, r *http.Request) {
	data, err := global.AssetManager.GetAssetString("assets/backups.js")
	if err != nil {
		log.Fatalf("Failed to read asset: %v", err)
	}
	w.Header().Set("Content-Type", "text/javascript")
	fmt.Fprintf(w, "%s", data)
}

func HandleSomething(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Something else")
}
