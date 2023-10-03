package rabbitmq

import (
	"github.com/programme-lv/tester/internal/messaging"
	"github.com/programme-lv/tester/internal/testing"
)

func (r *Gatherer) StartEvaluation(testerInfo string) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.SubmissionReceived,
		Data: messaging.SubmissionReceivedData{
			TestEnvInfo: testerInfo,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) StartTesting(maxScore int) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestingStarted,
		Data: messaging.TestingStartedData{
			MaxScore: maxScore,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) IncrementScore(delta int) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.IncrementScore,
		Data: messaging.IncrementScoreData{
			Delta: delta,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishWithInternalServerError(err error) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.FinishEvaluation,
		Data: messaging.FinishEvaluationData{
			Err: err,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishEvaluation() {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.FinishEvaluation,
		Data: messaging.FinishEvaluationData{
			Err: nil,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) StartCompilation() {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.CompilationStarted,
		Data:         messaging.CompilationStartedData{},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishCompilation(data testing.RuntimeData) {
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

func (r *Gatherer) IgnoreTest(testId int64) {
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

func (r *Gatherer) StartTest(testId int64) {
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestStarted,
		Data: messaging.TestStartedData{
			TestId: testId,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) ReportTestSubmissionRuntimeData(testId int64, rd testing.RuntimeData) {
	r.testRuntimeDataCache[testId].submissionRuntimeData = rd
}

func (r *Gatherer) ReportTestCheckerRuntimeData(testId int64, rd testing.RuntimeData) {
	r.testRuntimeDataCache[testId].checkerRuntimeData = rd
}

func toMessagingRuntimeData(rd *testing.RuntimeData) *messaging.RuntimeData {
	return &messaging.RuntimeData{
		Output: messaging.RuntimeOutput{
			Stdout:   rd.Output.Stdout,
			Stderr:   rd.Output.Stderr,
			ExitCode: rd.Output.ExitCode,
		},
		Metrics: messaging.RuntimeMetrics{
			CpuTimeMillis:  rd.Metrics.CpuTimeMillis,
			WallTimeMillis: rd.Metrics.WallTimeMillis,
			MemoryKBytes:   rd.Metrics.MemoryKBytes,
		},
	}
}

func (r *Gatherer) FinishTest(testId int64) {
	submissionRd := &r.testRuntimeDataCache[testId].submissionRuntimeData
	checkerRd := &r.testRuntimeDataCache[testId].checkerRuntimeData
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestFinished,
		Data: messaging.TestFinishedData{
			TestId:                testId,
			Verdict:               "",
			SubmissionRuntimeData: toMessagingRuntimeData(submissionRd),
			CheckerRuntimeData:    toMessagingRuntimeData(checkerRd),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishTestWithLimitExceeded(testId int64, flags testing.RuntimeExceededFlags) {
	submissionRd := &r.testRuntimeDataCache[testId].submissionRuntimeData
	checkerRd := &r.testRuntimeDataCache[testId].checkerRuntimeData
	verdict := messaging.IdlenessLimitExceeded
	if flags.TimeLimitExceeded {
		verdict = messaging.TimeLimitExceeded
	} else if flags.MemoryLimitExceeded {
		verdict = messaging.MemoryLimitExceeded
	}
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestFinished,
		Data: messaging.TestFinishedData{
			TestId:                testId,
			Verdict:               verdict,
			SubmissionRuntimeData: toMessagingRuntimeData(submissionRd),
			CheckerRuntimeData:    toMessagingRuntimeData(checkerRd),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishTestWithRuntimeError(testId int64) {
	submissionRd := &r.testRuntimeDataCache[testId].submissionRuntimeData
	checkerRd := &r.testRuntimeDataCache[testId].checkerRuntimeData
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestFinished,
		Data: messaging.TestFinishedData{
			TestId:                testId,
			Verdict:               messaging.RuntimeError,
			SubmissionRuntimeData: toMessagingRuntimeData(submissionRd),
			CheckerRuntimeData:    toMessagingRuntimeData(checkerRd),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishTestWithVerdictAccepted(testId int64) {
	submissionRd := &r.testRuntimeDataCache[testId].submissionRuntimeData
	checkerRd := &r.testRuntimeDataCache[testId].checkerRuntimeData
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestFinished,
		Data: messaging.TestFinishedData{
			TestId:                testId,
			Verdict:               messaging.Accepted,
			SubmissionRuntimeData: toMessagingRuntimeData(submissionRd),
			CheckerRuntimeData:    toMessagingRuntimeData(checkerRd),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishTestWithVerdictWrongAnswer(testId int64) {
	submissionRd := &r.testRuntimeDataCache[testId].submissionRuntimeData
	checkerRd := &r.testRuntimeDataCache[testId].checkerRuntimeData
	msg := &messaging.EvaluationResponse{
		FeedbackType: messaging.TestFinished,
		Data: messaging.TestFinishedData{
			TestId:                testId,
			Verdict:               messaging.WrongAnswer,
			SubmissionRuntimeData: toMessagingRuntimeData(submissionRd),
			CheckerRuntimeData:    toMessagingRuntimeData(checkerRd),
		},
	}
	r.sendEvalResponse(msg)
}
