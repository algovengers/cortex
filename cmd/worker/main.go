package main

import (
	"context"
	"cortex/internal"
	"cortex/internal/worker"
	"fmt"
)

func main() {
	qu, err := internal.GetQueue()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	msgs, err := qu.Consume()
	if err != nil {
		panic(err)
	}

	go func() {
		for d := range msgs {
			if err = worker.StartJob(ctx); err != nil {
				d.Nack(false, true)
			} else {
				d.Ack(false)
			}
		}
	}()

	fmt.Println("Worker Started!")
}
