package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/programme-lv/tester/internal/environment"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Submission struct {
	SourceCode string
	LanguageId string
}

type EvaluationRequest struct {
	TaskVersionId int
	Submission    Submission
}

type CorrelationStruct struct {
	IsEvaluation bool  `json:"is_evaluation"`
	EvaluationId int64 `json:"evaluation_id,omitempty"`
	UnixMillis   int64 `json:"unix_millis"`
	RandomInt63  int64 `json:"random_int_63"`
}

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

	_, err = ch.QueueDeclare(
		"eval_q",
		true,
		false,
		false,
		false,
		nil,
	)
	panicOnError(err)

	msg := EvaluationRequest{
		TaskVersionId: 1,
		Submission: Submission{
			SourceCode: "print(3)",
			LanguageId: "python3",
		},
	}

	marshalled, err := json.Marshal(msg)
	panicOnError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	correlation := CorrelationStruct{
		IsEvaluation: false,
		EvaluationId: 0,
		UnixMillis:   time.Now().UnixMilli(),
		RandomInt63:  rand.Int63(),
	}
	correlationJson, err := json.Marshal(correlation)
	panicOnError(err)

	log.Println("Publishing message...")
	err = ch.PublishWithContext(
		ctx,
		"",
		"eval_q",
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: string(correlationJson),
			ReplyTo:       "res_q",
			Body:          marshalled,
		})
	panicOnError(err)
	log.Println("Message published")

	log.Println("Exiting...")
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
