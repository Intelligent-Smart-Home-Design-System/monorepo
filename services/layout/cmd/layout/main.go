package main

import (
	"fmt"
	"log"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/configs"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/entities"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/rules/storage"
)

// TODO: убрать
func GetApartment() *entities.Apartment {
	return &entities.Apartment{}
}

// TODO: убрать
func GetSelectedLevels() map[string]string {
	return make(map[string]string)
}

func main() {
	apartment := GetApartment()
	selectedLevels := GetSelectedLevels()
	storage := storage.NewStorage()

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

	// TODO: записать результаты на плане квартиры
}