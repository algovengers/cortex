package internal

import (
	"fmt"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

const mainQueueName = "main-queue"

type JobQueue struct {
	channel *amqp.Channel
	queue   amqp.Queue
}

func (qu *JobQueue) Push(id string) (bool, error) {
	err := qu.channel.Publish(
		"",
		id,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(id),
		},
	)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (qu *JobQueue) Consume() (<-chan amqp.Delivery, error) {
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

	q, err := ch.QueueDeclare(mainQueueName, false, false, false, false, nil)
	if err != nil {
		fmt.Printf("Error creating a queue %v", err)
		panic(err)
	}

	return &JobQueue{
		queue:   q,
		channel: ch,
	}, nil

}
