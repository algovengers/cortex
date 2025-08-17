package main

import (
	"context"
	"cortex/internal"
	"cortex/internal/api"
	"cortex/internal/db"
	"log"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	ctx := context.Background()

	if port == "" {
		log.Fatal("PORT not found")
		os.Exit(0)
	}

	dbUrl := os.Getenv("DATABASE_URL")

	if dbUrl == "" {
		log.Fatal("DATABASE_URL not found")
		os.Exit(0)
	}

	db, err := db.Init(ctx, dbUrl)

	if err != nil {
		log.Fatal("Error connecting to db")
		os.Exit(0)
	}

	qu, err := internal.GetQueue()
	if err != nil {
		log.Fatal("Error getting the queue")
		os.Exit(0)
	}

	s := api.NewServer(db, qu)

	s.Start(port)
}
