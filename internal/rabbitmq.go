package internal

import (
	"context"
	"fmt"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const mainQueueName = "main-queue"

type JobQueue struct {
	channel *amqp.Channel
	queue   amqp.Queue
}

func (qu *JobQueue) Push(ctx context.Context, id string) error {
	return qu.channel.PublishWithContext(
		ctx,
		"",
		id,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(id),
			Timestamp:   time.Now(),
		},
	)
}

func (qu *JobQueue) Consume(ctx context.Context) (<-chan amqp.Delivery, error) {
	return qu.channel.Consume(mainQueueName, "", true, false, false, false, nil)
}

func GetQueue() (*JobQueue, error) {
	rabbitMqUrl := os.Getenv("RABBITMQ_URL")
	conn, err := amqp.Dial(rabbitMqUrl)

	if err != nil {
		return nil, fmt.Errorf("error creating a queue %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()

	if err != nil {
		return nil, fmt.Errorf("error creating a queue %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(mainQueueName, true, false, false, false, nil)
	if err != nil {
		fmt.Printf("Error creating a queue %v", err)
		panic(err)
	}

	return &JobQueue{
		queue:   q,
		channel: ch,
	}, nil

}
