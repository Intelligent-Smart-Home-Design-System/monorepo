package sender

// TODO: sender получает от engine map[string]any, преобразовывает в нужный формат (json / ...) и отдает другому сервису

type SimSender struct {
}

func NewSimSender() *SimSender {
	return &SimSender{}
}

func (s SimSender) Send() {
	panic("")
}
