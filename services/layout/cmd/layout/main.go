package main

import (
	"log"
	"net/http"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/handlers"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
)

func main() {
	tracksConfig, err := configs.LoadTracksConfig("internal/configs/tracks.json")
	if err != nil {
		log.Fatal("failed to load tracks config")
	}

	devicesConfig, err := configs.LoadDevicesConfig("internal/configs/tracks.json")
	if err != nil {
		log.Fatal("failed to load devices config")
	}

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules(devicesConfig)

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)

	handlers := handlers.NewHandlers()
	http.HandleFunc("/apartment", handlers.ApartmentHandler)   // TODO: договориться с названием URL
	http.HandleFunc("/layout", handlers.LayoutHandler(engine)) // TODO: договориться с названием URL

	err = http.ListenAndServe(":8080", nil)

	log.Println("Layout algorithm is running")
	log.Fatal(err)
}
