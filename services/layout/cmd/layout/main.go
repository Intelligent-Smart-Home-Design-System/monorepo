package main

import (
	"fmt"
	"log"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
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
	apartment := GetApartment()
	apartment.MakeRoomDependency()

	selectedLevels := GetSelectedLevels()

	storage := storage.NewStorage()
	storage.LoadAllSecurityRules()

	tracksConfig, err := configs.LoadTracksConfig("internal/configs/tracks.json")
	if err != nil {
		log.Fatal("failed to load tracks config")
	}

	devicesConfig, err := configs.LoadDevicesConfig("internal/configs/tracks.json")
	if err != nil {
		log.Fatal("failed to load devices config")
	}

	engine := engine.NewEngine(storage, tracksConfig, devicesConfig)

	_, err = engine.PlaceDevices(apartment, selectedLevels) // вся расстановка в квартире
	if err != nil {
		_ = fmt.Errorf("failed to place devices: %w", err)
	}

	go func() {
		// TODO: несколько пользователей (параллельно)
	}()

	// TODO: записать результаты на плане квартиры
}
