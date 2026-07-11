package ws

import (
	"net/http"
	"sync"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/client"
	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "" ||
				origin == "http://localhost:3000" ||
				origin == "http://127.0.0.1:3000"
		},
	}
)

// Manager представляет структуру менеджера для управления WebSocket клиентами и их взаимодействием с симуляцией.
type Manager struct {
	clients map[*Client]bool
	mu      sync.RWMutex

	simService client.SimulationService
}

// NewManager создает новый экземпляр Manager с инициализацией необходимых полей.
func NewManager(simService client.SimulationService) *Manager {
	return &Manager{
		clients:    make(map[*Client]bool),
		simService: simService,
	}
}

// ServeWS обрабатывает входящие HTTP запросы и устанавливает WebSocket соединение с клиентом. После успешного подключения,
// создается новый клиент и запускаются горутины для чтения и записи сообщений.
func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}

	client := NewClient(conn, m, m.simService)
	m.addClient(client)

	go client.ReadMessages()
	go client.WriteMessages()
}

// addClient добавляет нового клиента в список активных клиентов, обеспечивая безопасность доступа с помощью мьютекса.
func (m *Manager) addClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[client] = true
}

// removeClient удаляет клиента из списка активных клиентов и закрывает его соединение, обеспечивая безопасность доступа с помощью мьютекса.
func (m *Manager) removeClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[client]; exists {
		client.connection.Close()
		delete(m.clients, client)
	}
}
