package rmqgath

import (
	"context"
	"encoding/json"
	"time"

	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/pkg/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

type testRuntimeData struct {
	submissionRuntimeData models.RuntimeData
	checkerRuntimeData    models.RuntimeData
}

type Gatherer struct {
	amqpChannel          *amqp.Channel
	replyTo              string
	testRuntimeDataCache map[int64]*testRuntimeData
}

var _ testing.EvalResGatherer = (*Gatherer)(nil)

func NewRabbitMQGatherer(ch *amqp.Channel, replyTo string) *Gatherer {
	return &Gatherer{
		amqpChannel:          ch,
		replyTo:              replyTo,
		testRuntimeDataCache: make(map[int64]*testRuntimeData),
	}
}

func (r *Gatherer) declareReplyToQueue() {
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

func (r *Gatherer) sendEvalResponse(msg *messaging.EvaluationResponse) {
	r.declareReplyToQueue()

	marshalled, err := json.Marshal(msg)
	panicOnError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	correlationJson, err := json.Marshal(r.correlation)
	panicOnError(err)

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
