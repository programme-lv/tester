package rmqgath

import (
	"encoding/json"

	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/pkg/messaging"
	"github.com/wagslane/go-rabbitmq"
)

type testRuntimeData struct {
	submissionRuntimeData models.RuntimeData
	checkerRuntimeData    models.RuntimeData
}

type Gatherer struct {
	publisher            *rabbitmq.Publisher
	replyTo              string
	testRuntimeDataCache map[int64]*testRuntimeData
}

var _ testing.EvalResGatherer = (*Gatherer)(nil)

func NewRabbitMQGatherer(conn *rabbitmq.Conn, replyTo string) *Gatherer {
	publisher, err := rabbitmq.NewPublisher(conn)
	panicOnError(err)

	return &Gatherer{
		publisher:            publisher,
		replyTo:              replyTo,
		testRuntimeDataCache: make(map[int64]*testRuntimeData),
	}
}

func (r *Gatherer) sendEvalResponse(msg *messaging.EvaluationResponse) {
	marshalled, err := json.Marshal(msg)
	panicOnError(err)

	err = r.publisher.Publish(
		marshalled,
		[]string{r.replyTo},
		rabbitmq.WithPublishOptionsContentType("application/json"),
	)

	panicOnError(err)
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
