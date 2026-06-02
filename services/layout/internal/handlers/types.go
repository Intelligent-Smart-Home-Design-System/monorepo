package handlers

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"

type ApartmentRequest struct {
	SessionID string               `json:"session_id"`
	Apartment *apartment.Apartment `json:"apartment"`
}

type PlacementRequest struct {
	SessionID string            `json:"session_id"`
	Levels    map[string]string `json:"levels"`
}

type PlacementResponse struct {
	SessionID string            `json:"session_id"`
	Layout    *apartment.Layout `json:"layout"`
}
