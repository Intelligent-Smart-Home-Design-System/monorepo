package sender

import (
	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/api"
)

type Sender interface {
	// Run запускает SimSender, который слушает канал EventsChan и отправляет события при их поступлении.
	Run()

	// AddEvent добавляет событие в канал EventsChan для отправки.
	AddEvent(dto api.EventOutDTO)

	// Send отправляет событие в другой сервис.
	Send(dto api.EventOutDTO)
}
