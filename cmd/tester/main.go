package main

import (
	"encoding/json"

	"log"

	"github.com/programme-lv/tester/internal/gatherers/rmqgath"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/pkg/messaging"

	_ "github.com/lib/pq"
	"github.com/programme-lv/tester/internal/environment"
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
		request := messaging.EvaluationRequest{}
		err := json.Unmarshal(d.Body, &request)
		panicOnError(err)

		correlation := messaging.ResponseCorrelation{}
		err = json.Unmarshal([]byte(d.CorrelationId), &correlation)
		panicOnError(err)

		rmqGatherer := rmqgath.NewRabbitMQGatherer(ch, correlation, d.ReplyTo)

		err = testing.EvaluateSubmission(request, rmqGatherer)
		panicOnError(err)

		err = d.Ack(false)
		panicOnError(err)
	}

	log.Println("Exiting...")
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
