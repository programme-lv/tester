package rabbitmq

import (
	"github.com/programme-lv/tester/internal/messaging"
	"github.com/programme-lv/tester/internal/testing"
)

func (r *RabbitMQGatherer) StartEvaluation(testerInfo string) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.SubmissionReceived,
		Data: messaging.SubmissionReceivedData{
			TestEnvInfo: testerInfo,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *RabbitMQGatherer) StartTesting(maxScore int) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestingStarted,
		Data: messaging.TestingStartedData{
			MaxScore: maxScore,
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

func (r *RabbitMQGatherer) IgnoreTest(testId int64) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestFinished,
		Data: messaging.TestFinishedData{
			TestId:                testId,
			Verdict:               messaging.Ignored,
			SubmissionRuntimeData: nil,
			CheckerRuntimeData:    nil,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *RabbitMQGatherer) StartTest(testId int64) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestStarted,
		Data: messaging.TestStartedData{
			TestId: testId,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *RabbitMQGatherer) ReportTestSubmissionRuntimeData(testId int64, rd testing.RuntimeData) {
	r.testRuntimeDataCache[testId].submissionRuntimeData = rd
}

func (r *RabbitMQGatherer) ReportTestCheckerRuntimeData(testId int64, rd testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) FinishTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) FinishTestWithLimitExceeded(testId int64, flags testing.RuntimeExceededFlags) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) FinishTestWithRuntimeError(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) FinishTestWithVerdictAccepted(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r *RabbitMQGatherer) FinishTestWithVerdictWrongAnswer(testId int64) {
	//TODO implement me
	panic("implement me")
}
