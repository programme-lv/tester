package rabbitmq

import (
	"context"
	"encoding/json"
	"github.com/programme-lv/tester/internal/messaging"
	"github.com/programme-lv/tester/internal/testing"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
)

type testRuntimeData struct {
	submissionRuntimeData testing.RuntimeData
	checkerRuntimeData    testing.RuntimeData
}

type RabbitMQGatherer struct {
	amqpChannel          *amqp.Channel
	correlation          messaging.Correlation
	replyTo              string
	testRuntimeDataCache map[int64]*testRuntimeData
}

var _ testing.EvalResGatherer = (*RabbitMQGatherer)(nil)

func NewRabbitMQGatherer(ch *amqp.Channel, correlation messaging.Correlation, replyTo string) *RabbitMQGatherer {
	return &RabbitMQGatherer{
		amqpChannel:          ch,
		correlation:          correlation,
		replyTo:              replyTo,
		testRuntimeDataCache: make(map[int64]testRuntimeData),
	}
}

func (r *RabbitMQGatherer) declareReplyToQueue() {
	_, err := r.amqpChannel.QueueDeclare(
		r.replyTo,
		true,
		false,
		false,
		false,
		nil,
	)
	panicOnError(err)
}

func (r *RabbitMQGatherer) sendEvalResponse(msg *messaging.EvaluationResponse) {
	r.declareReplyToQueue()

	marshalled, err := json.Marshal(msg)
	panicOnError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	correlationJson, err := json.Marshal(r.correlation)
	panicOnError(err)

	log.Println("Publishing message...")
	err = r.amqpChannel.PublishWithContext(
		ctx,
		"",
		r.replyTo,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: string(correlationJson),
			Body:          marshalled,
		})
	panicOnError(err)
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
