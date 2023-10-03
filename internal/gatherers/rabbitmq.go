package gatherers

import (
	"context"
	"encoding/json"
	"github.com/programme-lv/tester/internal/messaging"
	"github.com/programme-lv/tester/internal/testing"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"math/rand"
	"time"
)

type RabbitMQGatherer struct {
	amqpChannel *amqp.Channel
	correlation messaging.Correlation
	replyTo     string
}

var _ testing.EvalResGatherer = (*RabbitMQGatherer)(nil)

func NewRabbitMQGatherer(ch *amqp.Channel, correlation messaging.Correlation, replyTo string) *RabbitMQGatherer {
	return &RabbitMQGatherer{correlation: correlation, replyTo: replyTo}
}

func (r RabbitMQGatherer) StartEvaluation(testerInfo string, evalMaxScore int) {
	response := messaging.EvaluationResponse{
		FeedbackType: "123",
		Data:         nil,
	}
	// ensure reply to queue exists

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
			SourceCode: "print(3)",
			LanguageId: "python3",
		},
	}

	marshalled, err := json.Marshal(msg)
	panicOnError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	correlation := messaging.Correlation{
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
	log.Println("Message published") //TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) IncrementScore(delta int) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) FinishWithInternalServerError(err error) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) FinishEvaluation() {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) StartCompilation() {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) FinishCompilation(data testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) StartTesting() {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) IgnoreTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) StartTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestUserRuntimeData(testId int64, rd testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestUserRuntimeLimitExceeded(testId int64, flags testing.RuntimeExceededFlags) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestUserRuntimeError(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestCheckerRuntimeData(testId int64, rd testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestVerdictAccepted(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestVerdictWrongAnswer(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) FinishTest(testId int64) {
	//TODO implement me
	panic("implement me")
}
