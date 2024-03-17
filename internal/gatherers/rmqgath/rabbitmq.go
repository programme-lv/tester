package rmqgath

import (
	"log"

	"github.com/klauspost/compress/snappy"
	"github.com/programme-lv/director/msg"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/wagslane/go-rabbitmq"
	"google.golang.org/protobuf/proto"
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
	log.Printf("m: %+v", m)
	marshalled, err := proto.Marshal(m)
	panicOnError(err)
	log.Printf("marshalled: %+v", marshalled)

	compressed := snappy.Encode(nil, marshalled)
	log.Printf("compressed: %+v", compressed)

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
