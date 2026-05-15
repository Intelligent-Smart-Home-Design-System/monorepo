package handlers

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/apartment"
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/layout/internal/events/engine"
)

type Handlers struct {
	mu             sync.RWMutex
	sessionApartment map[string]*apartment.Apartment
}

func NewHandlers() *Handlers {
	return &Handlers{sessionApartment: make(map[string]*apartment.Apartment)}
}

// ApartmentHandler - обработчик парсинга квартиры
func (h *Handlers) ApartmentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method, accepted only POST method", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var req ApartmentRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON struct", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "Miss session id", http.StatusBadRequest)
		return
	}

	if req.Apartment == nil {
		http.Error(w, "Invalid apartment struct (is nil)", http.StatusBadRequest)
		return
	}

	req.Apartment.Index()

	h.mu.Lock()
	h.sessionApartment[req.SessionID] = req.Apartment
	h.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// LayoutHandler - обработчик отправки данных
func (h *Handlers) LayoutHandler(eng *engine.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid method, accepted only POST method", http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		var req PlacementRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request struct", http.StatusBadRequest)
			return
		}

		if req.SessionID == "" {
			http.Error(w, "Miss session id", http.StatusBadRequest)
			return
		}

		h.mu.RLock()
		apart, ok := h.sessionApartment[req.SessionID]
		h.mu.RUnlock()

		if !ok || apart == nil {
			message := "Failed to load apartment for session " + req.SessionID
			http.Error(w, message, http.StatusBadRequest)
			return
		}

		layout, err := eng.PlaceDevices(apart, req.Levels)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := PlacementResponse{
			SessionID: req.SessionID,
			Layout:    layout,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
