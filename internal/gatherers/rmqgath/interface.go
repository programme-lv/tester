package rmqgath

import (
	"log"

	"github.com/programme-lv/tester/internal/testing/models"
	messaging2 "github.com/programme-lv/tester/pkg/messaging"
	"github.com/programme-lv/tester/pkg/messaging/statuses"
)

func (r *Gatherer) StartEvaluation() {
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.UpdateEvalStatus,
		Data: messaging2.UpdateEvalStatusData{
			Status: statuses.Received,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) StartTesting(maxScore int64) {
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.UpdateEvalStatus,
		Data: messaging2.UpdateEvalStatusData{
			Status: statuses.Testing,
		},
	}
	r.sendEvalResponse(msg)

	msg = &messaging2.EvaluationResponse{
		FeedbackType: messaging2.SetMaxScore,
		Data: messaging2.SetMaxScoreData{
			MaxScore: int(maxScore),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) IncrementScore(delta int64) {
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.IncrementScore,
		Data: messaging2.IncrementScoreData{
			Delta: int(delta),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishWithInternalServerError(err error) {
	log.Printf("Internal server error: %v", err)
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.UpdateEvalStatus,
		Data: messaging2.UpdateEvalStatusData{
			Status: statuses.InternalServerError,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishEvaluation() {
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.UpdateEvalStatus,
		Data: messaging2.UpdateEvalStatusData{
			Status: statuses.Finished,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) StartCompilation() {
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.UpdateEvalStatus,
		Data: messaging2.UpdateEvalStatusData{
			Status: statuses.Compiling,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishCompilation(data *models.RuntimeData) {
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.CompilationFinished,
		Data: messaging2.CompilationFinishedData{
			RuntimeData: toMessagingRuntimeData(data),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishWithCompilationError() {
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.UpdateEvalStatus,
		Data: messaging2.UpdateEvalStatusData{
			Status: statuses.CompilationError,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) IgnoreTest(testId int64) {
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.TestFinished,
		Data: messaging2.TestFinishedData{
			TestId:                testId,
			Verdict:               statuses.Ignored,
			SubmissionRuntimeData: nil,
			CheckerRuntimeData:    nil,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) StartTest(testId int64) {
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.TestStarted,
		Data: messaging2.TestStartedData{
			TestId: testId,
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) ReportTestSubmissionRuntimeData(testId int64, rd *models.RuntimeData) {
	if r.testRuntimeDataCache[testId] == nil {
		r.testRuntimeDataCache[testId] = &testRuntimeData{}
	}
	r.testRuntimeDataCache[testId].submissionRuntimeData = *rd
}

func (r *Gatherer) ReportTestCheckerRuntimeData(testId int64, rd *models.RuntimeData) {
	if r.testRuntimeDataCache[testId] == nil {
		r.testRuntimeDataCache[testId] = &testRuntimeData{}
	}
	r.testRuntimeDataCache[testId].checkerRuntimeData = *rd
}

func toMessagingRuntimeData(rd *models.RuntimeData) *messaging2.RuntimeData {
	return &messaging2.RuntimeData{
		Output: messaging2.RuntimeOutput{
			Stdout:   rd.Output.Stdout,
			Stderr:   rd.Output.Stderr,
			ExitCode: rd.Output.ExitCode,
		},
		Metrics: messaging2.RuntimeMetrics{
			CpuTimeMillis:  rd.Metrics.CpuTimeMillis,
			WallTimeMillis: rd.Metrics.WallTimeMillis,
			MemoryKBytes:   rd.Metrics.MemoryKBytes,
		},
	}
}

func (r *Gatherer) FinishTest(testId int64) {
	submissionRd := &r.testRuntimeDataCache[testId].submissionRuntimeData
	checkerRd := &r.testRuntimeDataCache[testId].checkerRuntimeData
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.TestFinished,
		Data: messaging2.TestFinishedData{
			TestId:                testId,
			Verdict:               "",
			SubmissionRuntimeData: toMessagingRuntimeData(submissionRd),
			CheckerRuntimeData:    toMessagingRuntimeData(checkerRd),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishTestWithLimitExceeded(testId int64, flags models.RuntimeExceededFlags) {
	submissionRd := &r.testRuntimeDataCache[testId].submissionRuntimeData
	checkerRd := &r.testRuntimeDataCache[testId].checkerRuntimeData
	verdict := statuses.IdlenessLimitExceeded
	if flags.TimeLimitExceeded {
		verdict = statuses.TimeLimitExceeded
	} else if flags.MemoryLimitExceeded {
		verdict = statuses.MemoryLimitExceeded
	}
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.TestFinished,
		Data: messaging2.TestFinishedData{
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
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.TestFinished,
		Data: messaging2.TestFinishedData{
			TestId:                testId,
			Verdict:               statuses.RuntimeError,
			SubmissionRuntimeData: toMessagingRuntimeData(submissionRd),
			CheckerRuntimeData:    toMessagingRuntimeData(checkerRd),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishTestWithVerdictAccepted(testId int64) {
	submissionRd := &r.testRuntimeDataCache[testId].submissionRuntimeData
	checkerRd := &r.testRuntimeDataCache[testId].checkerRuntimeData
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.TestFinished,
		Data: messaging2.TestFinishedData{
			TestId:                testId,
			Verdict:               statuses.Accepted,
			SubmissionRuntimeData: toMessagingRuntimeData(submissionRd),
			CheckerRuntimeData:    toMessagingRuntimeData(checkerRd),
		},
	}
	r.sendEvalResponse(msg)
}

func (r *Gatherer) FinishTestWithVerdictWrongAnswer(testId int64) {
	submissionRd := &r.testRuntimeDataCache[testId].submissionRuntimeData
	checkerRd := &r.testRuntimeDataCache[testId].checkerRuntimeData
	msg := &messaging2.EvaluationResponse{
		FeedbackType: messaging2.TestFinished,
		Data: messaging2.TestFinishedData{
			TestId:                testId,
			Verdict:               statuses.WrongAnswer,
			SubmissionRuntimeData: toMessagingRuntimeData(submissionRd),
			CheckerRuntimeData:    toMessagingRuntimeData(checkerRd),
		},
	}
	r.sendEvalResponse(msg)
}
