package sender

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
)

const maxEventsBuffer = 10000

// SimSender реализует интерфейс Sender для отправки данных о событиях в другие сервисы.
type SimSender struct {
	EventsChan chan api.EventOutDTO
}

// NewSimSender создает SimSender.
func NewSimSender() *SimSender {
	return &SimSender{
		EventsChan: make(chan api.EventOutDTO, maxEventsBuffer),
	}
}

// Run запускает SimSender, который слушает канал EventsChan и отправляет события при их поступлении.
func (s *SimSender) Run() {
	for event := range s.EventsChan {
		s.Send(event)
	}
}

// AddEvent добавляет событие в канал EventsChan для отправки.
func (s *SimSender) AddEvent(OutDTO api.EventOutDTO) {
	s.EventsChan <- OutDTO
}

// Send отправляет событие в другой сервис.
func (s *SimSender) Send(OutDTO api.EventOutDTO) {
	panic("todo")
}
