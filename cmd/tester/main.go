package main

import (
	"log"

	"github.com/golang/snappy"
	"github.com/programme-lv/director/msg"
	"google.golang.org/protobuf/proto"

	_ "github.com/lib/pq"
	"github.com/programme-lv/tester/internal/environment"
	"github.com/programme-lv/tester/internal/gatherers/rmqgath"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testing/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	cfg := environment.ReadEnvConfig()

	rabbit, err := amqp.Dial(cfg.AMQPConnString)
	panicOnError(err)
	defer func(rabbit *amqp.Connection) {
		err := rabbit.Close()
		panicOnError(err)
	}(rabbit)

	ch, err := openChannel(rabbit)
	panicOnError(err)
	defer func(ch *amqp.Channel) {
		err := ch.Close()
		panicOnError(err)
	}(ch)

	q, err := declareEvalQueue(ch)
	panicOnError(err)

	msgs, err := startConsuming(ch, q)
	panicOnError(err)

	for d := range msgs {
		decompressed, err := snappy.Decode(nil, d.Body)
		panicOnError(err)
		request := msg.EvaluationRequest{}
		err = proto.Unmarshal(decompressed, &request)
		panicOnError(err)

		rmqGatherer := rmqgath.NewRabbitMQGatherer(ch, d.ReplyTo)

		reqModel := translateMsgRequestToTestingModel(&request)
		err = testing.EvaluateSubmission(&reqModel, rmqGatherer)
		panicOnError(err)

		err = d.Ack(false)
		panicOnError(err)
	}

	log.Println("Exiting...")
}

func translateMsgRequestToTestingModel(request *msg.EvaluationRequest) models.EvaluationRequest {
	result := models.EvaluationRequest{
		Submission: request.Submission,
		PLanguage: models.PLanguage{
			ID:               request.Language.Id,
			FullName:         request.Language.Name,
			CodeFilename:     request.Language.CodeFilename,
			CompileCmd:       request.Language.CompileCmd,
			CompiledFilename: request.Language.CompiledFilename,
			ExecCmd:          request.Language.ExecuteCmd,
		},
		Limits: models.Limits{
			CPUTimeMillis: int(request.Limits.CPUTimeMillis),
			MemKibibytes:  int(request.Limits.MemKibiBytes),
		},
		EvalTypeID:     request.EvalType,
		Tests:          nil,
		Subtasks:       nil,
		TestlibChecker: request.TestlibChecker,
	}

	tests := make([]models.TestRef, len(request.Tests))
	for i, test := range request.Tests {
		tests[i] = models.TestRef{
			ID:          int(test.Id),
			InContent:   test.InContent,
			InSHA256:    test.InSha256,
			InDownlUrl:  test.InDownloadUrl,
			AnsContent:  test.AnsContent,
			AnsSHA256:   test.AnsSha256,
			AnsDownlUrl: test.AnsDownloadUrl,
		}
	}
	result.Tests = tests

	// TODO: subtask support
	// subtasks := make([]models.Subtask, len(request.Subtasks))
	// for i, subtask := range request. {
	// 	subtasks[i] = models.Subtask{
	// 		ID:      int(subtask.Id),
	// 		Score:   int(subtask.Score),
	// 		TestIDs: subtask.TestIds,
	// 	}
	// }
	// result.Subtasks = subtasks

	return result
}

func openChannel(rabbit *amqp.Connection) (*amqp.Channel, error) {
	ch, err := rabbit.Channel()
	if err != nil {
		return nil, err
	}

	prefetchCount := 1 // process one message at a time
	prefetchSize := 0  // don't limit the size of the message
	global := false    // apply the settings to the current channel only
	err = ch.Qos(prefetchCount, prefetchSize, global)
	if err != nil {
		return nil, err
	}

	return ch, nil
}

func declareEvalQueue(ch *amqp.Channel) (amqp.Queue, error) {
	durable := true     // queue will survive broker restarts
	autoDelete := false // queue won't be deleted once the connection is closed
	exclusive := false  // queue can be accessed by other connections
	noWait := false     // don't wait for the server to confirm the queue creation
	args := make(amqp.Table)
	return ch.QueueDeclare("eval_q", durable, autoDelete, exclusive, noWait, args)
}

func startConsuming(ch *amqp.Channel, q amqp.Queue) (<-chan amqp.Delivery, error) {
	consumer := ""     // generate a unique consumer name
	autoAck := false   // don't automatically acknowledge the messages
	exclusive := false // queue can be accessed by other connections
	noLocal := false   // don't deliver own messages
	noWait := false    // don't wait for the server to confirm the consumer creation
	args := make(amqp.Table)
	return ch.Consume(q.Name, consumer, autoAck, exclusive, noLocal, noWait, args)
}

func panicOnError(err error) {
	if err != nil {
		log.Panic(err)
	}
}
