package main

import (
	"context"
	"encoding/json"
	"github.com/programme-lv/tester/internal/messaging"
	"log"
	"math/rand"
	"time"

	_ "github.com/lib/pq"
	"github.com/programme-lv/tester/internal/environment"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg := environment.ReadEnvConfig()

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

	msg := messaging.EvaluationRequest{
		TaskVersionId: 1,
		Submission: messaging.Submission{
			SourceCode: exampleCpp,
			LanguageId: "cpp17",
		},
	}

	marshalled, err := json.Marshal(msg)
	panicOnError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	correlation := messaging.Correlation{
		HasEvaluationId: false,
		EvaluationId:    0,
		UnixMillis:      time.Now().UnixMilli(),
		RandomInt63:     rand.Int63(),
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
