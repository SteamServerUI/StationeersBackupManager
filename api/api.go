package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/SteamServerUI/StationeersBackupManagerPlugin/global"
)

func HandleSomethingElse(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Something else")
}

func HandleTextFromAssetsManager(w http.ResponseWriter, r *http.Request) {
	data, err := global.AssetManager.GetAssetString("assets/index.html")
	if err != nil {
		log.Fatalf("Failed to read asset: %v", err)
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "%s", data)
}
func HandleBinaryFromAssetsManager(w http.ResponseWriter, r *http.Request) {
	data, err := global.AssetManager.GetAsset("assets/image.png")
	if err != nil {
		log.Fatalf("Failed to read asset: %v", err)
	}
	_, err = w.Write(data)
	if err != nil {
		log.Printf("Failed to write response: %v", err)
		http.Error(w, "Failed to serve binary", http.StatusInternalServerError)
		return
	}
}
