package main

import (
	"context"
	"log"
	"timetable-homework-tgbot/internal/infrastracture/database"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("key.env")

	ctx := context.Background()
	db, err := database.NewDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Migrate(ctx); err != nil {
		log.Fatal(err)
	}

	if err := db.FillDatabase(); err != nil {
		log.Fatal(err)
	}
}
