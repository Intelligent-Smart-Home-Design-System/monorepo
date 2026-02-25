package activities

import (
	"context"
	"fmt"
	"time"
)

type GreetingActivity struct{}

type GreetRequest struct {
	Name string
}

type GreetResponse struct {
	Message   string
	Timestamp time.Time
}

func (a *GreetingActivity) SayHello(ctx context.Context, req GreetRequest) (*GreetResponse, error) {
	message := fmt.Sprintf("Привет, %s! Добро пожаловать в Temporal!", req.Name)
	
	return &GreetResponse{
		Message:   message,
		Timestamp: time.Now(),
	}, nil
}

func (a *GreetingActivity) ProcessData(ctx context.Context, data string) (string, error) {
	time.Sleep(2 * time.Second)
	
	result := fmt.Sprintf("Обработано: %s (длина: %d символов)", data, len(data))
	return result, nil
}

func (a *GreetingActivity) SendNotification(ctx context.Context, message string) error {
	fmt.Printf("Отправка уведомления: %s\n", message)
	time.Sleep(1 * time.Second)
	fmt.Println("Уведомление отправлено успешно")
	return nil
}
