package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/programme-lv/tester/internal/database"
	"github.com/programme-lv/tester/internal/environment"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg := environment.ReadEnvConfig()

	log.Println("Connecting to Postgres...")
	postgres, err := sqlx.Connect("postgres", cfg.SqlxConnString)
	panicOnError(err)
	defer postgres.Close()
	log.Println("Connected to Postgres")

	log.Println("Connecting to RabbitMQ...")
	rabbit, err := amqp.Dial(cfg.AMQPConnString)
	panicOnError(err)
	defer rabbit.Close()
	log.Println("Connected to RabbitMQ")

	// TODO: submit all published task versions
	publishedTaskVersions, err := database.SelectPublishedTaskVersions(postgres)
	panicOnError(err)

	for _, taskVersion := range publishedTaskVersions {
		log.Printf("version: %v\n", taskVersion)
	}

	log.Println("Exiting...")
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
