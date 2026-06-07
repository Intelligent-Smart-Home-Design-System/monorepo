<<<<<<< HEAD
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
	err := configs.LoadTracksConfig("internal/configs/tracks.json")
	if err != nil {
		log.Fatal("failed to load tracks config")
	}

	err = configs.LoadDevicesConfig("internal/configs/tracks.json")
	if err != nil {
		log.Fatal("failed to load devices config")
	}

	storage := storage.NewStorage()
	storage.LoadAllRules()

	engine := engine.NewEngine(storage)

	handlers := handlers.NewHandlers()
	http.HandleFunc("/apartment", handlers.ApartmentHandler)   // TODO: договориться с названием URL
	http.HandleFunc("/layout", handlers.LayoutHandler(engine)) // TODO: договориться с названием URL

	err = http.ListenAndServe(":8080", nil)

	log.Println("Layout algorithm is running")
	log.Fatal(err)
}
=======
package main

import (
	"fmt"
	"log"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/exporter"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
)

// TODO: убрать
func GetApartment() *apartment.Apartment {
	return &apartment.Apartment{}
}

// TODO: убрать
func GetSelectedLevels() map[string]string {
	return make(map[string]string)
}

func main() {
	apartmentStruct := GetApartment()
	apartmentStruct.Index()

	selectedLevels := GetSelectedLevels()

	err := configs.LoadTracksConfig("internal/configs/tracks.json")
	if err != nil {
		log.Fatal("failed to load tracks config")
	}

	err = configs.LoadDevicesConfig("internal/configs/devices.json")
	if err != nil {
		log.Fatal("failed to load devices config")
	}

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()
	storage.LoadAllLightingRules()
	storage.LoadAllClimateRules()

	engine := engine.NewEngine(storage)

	layout, err := engine.PlaceDevices(apartmentStruct, selectedLevels) // вся расстановка в квартире
	if err != nil {
		_ = fmt.Errorf("failed to place devices: %w", err)
	}

	outputJSON, err := exporter.ExportToJSON(layout)
	if err != nil {
		_ = fmt.Errorf("failed to marshal output data")
	}

	fmt.Println(string(outputJSON))

	go func() {
		// TODO: несколько пользователей (параллельно)
	}()
}
>>>>>>> 4bf54f8 (hz)
