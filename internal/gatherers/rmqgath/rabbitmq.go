package rmqgath

import (
	"github.com/gogo/protobuf/proto"
	"github.com/klauspost/compress/snappy"
	"github.com/programme-lv/director/msg"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testing/models"
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

func (r *Gatherer) sendEvalResponse(m *msg.EvaluationFeedback) {
	marshalled, err := proto.Marshal(m)
	panicOnError(err)

	compressed := snappy.Encode(nil, marshalled)

	err = r.publisher.Publish(
		compressed,
		[]string{r.replyTo},
		rabbitmq.WithPublishOptionsContentType("application/octet-stream"),
	)
	panicOnError(err)
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
