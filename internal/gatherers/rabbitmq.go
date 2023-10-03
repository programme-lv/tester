package gatherers

import (
	"context"
	"encoding/json"
	"github.com/programme-lv/tester/internal/messaging"
	"github.com/programme-lv/tester/internal/testing"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
)

type RabbitMQGatherer struct {
	amqpChannel *amqp.Channel
	correlation messaging.Correlation
	replyTo     string
}

var _ testing.EvalResGatherer = (*RabbitMQGatherer)(nil)

func NewRabbitMQGatherer(ch *amqp.Channel, correlation messaging.Correlation, replyTo string) *RabbitMQGatherer {
	return &RabbitMQGatherer{
		amqpChannel: ch,
		correlation: correlation,
		replyTo:     replyTo,
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

func (r *RabbitMQGatherer) StartEvaluation(testerInfo string, evalMaxScore int) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.SubmissionReceived,
		Data: messaging.SubmissionReceivedData{
			TestEnvInfo: testerInfo,
			MaxScore:    evalMaxScore,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *RabbitMQGatherer) IncrementScore(delta int) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.IncrementScore,
		Data: messaging.IncrementScoreData{
			Delta: delta,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *RabbitMQGatherer) FinishWithInternalServerError(err error) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.FinishEvaluation,
		Data: messaging.FinishEvaluationData{
			Err: err,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *RabbitMQGatherer) FinishEvaluation() {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.FinishEvaluation,
		Data: messaging.FinishEvaluationData{
			Err: nil,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *RabbitMQGatherer) StartCompilation() {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.CompilationStarted,
		Data:         messaging.CompilationStartedData{},
	}
	r.sendEvalResponse(msg)
}

func (r *RabbitMQGatherer) FinishCompilation(data testing.RuntimeData) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.CompilationFinished,
		Data: messaging.CompilationFinishedData{
			RuntimeData: messaging.RuntimeData{
				Output: messaging.RuntimeOutput{
					Stdout: data.Output.Stdout,
					Stderr: data.Output.Stderr,
				},
				Metrics: messaging.RuntimeMetrics{
					CpuTimeMillis:  data.Metrics.CpuTimeMillis,
					WallTimeMillis: data.Metrics.WallTimeMillis,
					MemoryKBytes:   data.Metrics.MemoryKBytes,
				},
			},
		},
	}
	r.sendEvalResponse(msg)
}

func (r *RabbitMQGatherer) StartTesting() {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) IgnoreTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) StartTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) ReportTestUserRuntimeData(testId int64, rd testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) ReportTestUserRuntimeLimitExceeded(testId int64, flags testing.RuntimeExceededFlags) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) ReportTestUserRuntimeError(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) ReportTestCheckerRuntimeData(testId int64, rd testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) ReportTestVerdictAccepted(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) ReportTestVerdictWrongAnswer(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) FinishTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
