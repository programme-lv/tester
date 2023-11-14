package main

import (
	"encoding/json"
	"github.com/programme-lv/tester/internal/gatherers/pgdirect"

	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/pkg/messaging"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/programme-lv/tester/internal/environment"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg := environment.ReadEnvConfig()

	log.Println("Connecting to Postgres...")
	postgres, err := sqlx.Connect("postgres", cfg.SqlxConnString)
	panicOnError(err)
	defer func(postgres *sqlx.DB) {
		err := postgres.Close()
		panicOnError(err)
	}(postgres)
	log.Println("Connected to Postgres")

	log.Println("Connecting to RabbitMQ...")
	rabbit, err := amqp.Dial(cfg.AMQPConnString)
	panicOnError(err)
	defer func(rabbit *amqp.Connection) {
		err := rabbit.Close()
		panicOnError(err)
	}(rabbit)
	log.Println("Connected to RabbitMQ")

	ch, err := rabbit.Channel()
	panicOnError(err)
	defer func(ch *amqp.Channel) {
		err := ch.Close()
		panicOnError(err)
	}(ch)

	q, err := ch.QueueDeclare(
		"eval_q",
		true,
		false,
		false,
		false,
		nil,
	)
	panicOnError(err)

	err = ch.Qos(1, 0, false)
	panicOnError(err)

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	panicOnError(err)

	for d := range msgs {
		err = d.Ack(false)
		panicOnError(err)

		request := messaging.EvaluationRequest{}
		err := json.Unmarshal(d.Body, &request)
		panicOnError(err)
		log.Printf("Request: %+v", request)

		correlation := messaging.Correlation{}
		err = json.Unmarshal([]byte(d.CorrelationId), &correlation)
		panicOnError(err)
		log.Printf("Correlation: %+v", correlation)

		replyTo := d.ReplyTo
		log.Printf("ReplyTo: %s", replyTo)

		pgGatherer := pgdirect.NewPostgresGatherer(postgres, correlation.EvaluationId, correlation.RandomInt63)

		err = testing.EvaluateSubmission(request, pgGatherer, postgres)
		panicOnError(err)
	}

	log.Println("Exiting...")
}

func panicOnError(err error) {
	if err != nil {
		log.Panic(err)
	}
}
