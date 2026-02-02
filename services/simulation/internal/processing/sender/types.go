package sender

import "github.com/Intelligent-Smart-Home-Design-System/monorepo/services/simulation/internal/config"

type Sender interface {
	Send(dto api.EventOutDTO)
}
