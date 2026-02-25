package main

import (
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	
	"temporal-go-project/activities"
	"temporal-go-project/workflows"
)

const (
	TaskQueueName = "greeting-task-queue"
)

func main() {
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	log.Printf("Подключение к Temporal серверу: %s", temporalHost)

	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalln("Не удалось создать Temporal клиент:", err)
	}
	defer c.Close()

	w := worker.New(c, TaskQueueName, worker.Options{})

	w.RegisterWorkflow(workflows.GreetingWorkflow)
	w.RegisterWorkflow(workflows.SimpleWorkflow)

	greetingActivity := &activities.GreetingActivity{}
	w.RegisterActivity(greetingActivity.SayHello)
	w.RegisterActivity(greetingActivity.ProcessData)
	w.RegisterActivity(greetingActivity.SendNotification)

	log.Println("Worker запущен и ожидает задачи...")
	log.Printf("Task Queue: %s", TaskQueueName)
	
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Не удалось запустить worker:", err)
	}

	log.Println("Worker остановлен")
}
