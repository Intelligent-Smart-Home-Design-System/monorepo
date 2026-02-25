package main

import (
	"context"
	"log"
	"os"

	"go.temporal.io/sdk/client"
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

	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalln("Не удалось создать Temporal клиент:", err)
	}
	defer c.Close()

	workflowInput := workflows.GreetingWorkflowInput{
		Name: "Temporal User",
		Data: "Тестовые данные для обработки в Temporal Workflow",
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        "greeting-workflow-" + "1",
		TaskQueue: TaskQueueName,
	}

	log.Println("Запуск GreetingWorkflow...")
	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, workflows.GreetingWorkflow, workflowInput)
	if err != nil {
		log.Fatalln("Не удалось запустить workflow:", err)
	}

	log.Printf("Workflow запущен с ID: %s и RunID: %s", we.GetID(), we.GetRunID())

	var result workflows.GreetingWorkflowResult
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalln("Ошибка при выполнении workflow:", err)
	}

	log.Println("\nWorkflow успешно завершен!")
	log.Printf("Приветствие: %s", result.GreetingMessage)
	log.Printf("Обработанные данные: %s", result.ProcessedData)
	log.Printf("Время завершения: %s", result.CompletedAt.Format("2006-01-02 15:04:05"))

	log.Println("\nЗапуск SimpleWorkflow...")
	simpleOptions := client.StartWorkflowOptions{
		ID:        "simple-workflow-" + "1",
		TaskQueue: TaskQueueName,
	}

	we2, err := c.ExecuteWorkflow(context.Background(), simpleOptions, workflows.SimpleWorkflow, "Simple User")
	if err != nil {
		log.Fatalln("Не удалось запустить simple workflow:", err)
	}

	var simpleResult string
	err = we2.Get(context.Background(), &simpleResult)
	if err != nil {
		log.Fatalln("Ошибка при выполнении simple workflow:", err)
	}

	log.Printf("SimpleWorkflow результат: %s\n", simpleResult)
}
