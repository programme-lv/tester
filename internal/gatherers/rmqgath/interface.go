package rmqgath

import (
	"fmt"

	"github.com/programme-lv/director/msg"
	"github.com/programme-lv/tester/internal/testing/models"
)

func (r *Gatherer) StartEvaluation() {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_StartEvaluation{
			StartEvaluation: &msg.StartEvaluation{},
		},
	}

	r.sendEvalResponse(m)
}

func (r *Gatherer) StartTesting(maxScore int64) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_StartTesting{
			StartTesting: &msg.StartTesting{
				MaxScore: maxScore,
			},
		},
	}

	r.sendEvalResponse(m)
}

func (r *Gatherer) IncrementScore(delta int64) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_IncrementScore{
			IncrementScore: &msg.IncrementScore{
				Delta: delta,
			},
		},
	}

	r.sendEvalResponse(m)
}

func (r *Gatherer) FinishWithInternalServerError(err error) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_FinishWithInernalServerError{
			FinishWithInernalServerError: &msg.FinishWithInernalServerError{
				ErrorMsg: fmt.Sprintf("%+v", err),
			},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) FinishEvaluation() {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_FinishEvaluation{
			FinishEvaluation: &msg.FinishEvaluation{},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) StartCompilation() {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_StartCompilation{
			StartCompilation: &msg.StartCompilation{},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) FinishCompilation(data *models.RuntimeData) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_FinishCompilation{
			FinishCompilation: &msg.FinishCompilation{
				CompilationRData: &msg.RuntimeData{
					Stdout:         data.Output.Stdout,
					Stderr:         data.Output.Stderr,
					ExitCode:       data.Output.ExitCode,
					CpuTimeMillis:  data.Metrics.CpuTimeMillis,
					WallTimeMillis: data.Metrics.WallTimeMillis,
					MemKibiBytes:   data.Metrics.MemoryKBytes,
				},
			},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) FinishWithCompilationError() {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_FinishWithCompilationError{
			FinishWithCompilationError: &msg.FinishWithCompilationError{},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) IgnoreTest(testId int64) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_IgnoreTest{
			IgnoreTest: &msg.IgnoreTest{
				TestId: testId,
			},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) StartTest(testId int64) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_StartTest{
			StartTest: &msg.StartTest{
				TestId: testId,
			},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) ReportTestSubmissionRuntimeData(testId int64, rd *models.RuntimeData) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_ReportTestSubmissionRuntimeData{
			ReportTestSubmissionRuntimeData: &msg.ReportTestSubmissionRuntimeData{
				TestId: testId,
				RData: &msg.RuntimeData{
					Stdout:         rd.Output.Stdout,
					Stderr:         rd.Output.Stderr,
					ExitCode:       rd.Output.ExitCode,
					CpuTimeMillis:  rd.Metrics.CpuTimeMillis,
					WallTimeMillis: rd.Metrics.WallTimeMillis,
					MemKibiBytes:   rd.Metrics.MemoryKBytes,
				},
			},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) ReportTestCheckerRuntimeData(testId int64, rd *models.RuntimeData) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_ReportTestCheckerRuntimeData{
			ReportTestCheckerRuntimeData: &msg.ReportTestCheckerRuntimeData{
				TestId: testId,
				RData: &msg.RuntimeData{
					Stdout:         rd.Output.Stdout,
					Stderr:         rd.Output.Stderr,
					ExitCode:       rd.Output.ExitCode,
					CpuTimeMillis:  rd.Metrics.CpuTimeMillis,
					WallTimeMillis: rd.Metrics.WallTimeMillis,
					MemKibiBytes:   rd.Metrics.MemoryKBytes,
				},
			},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) FinishTestWithLimitExceeded(testId int64, flags models.RuntimeExceededFlags) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_FinishTestWithLimitExceeded{
			FinishTestWithLimitExceeded: &msg.FinishTestWithLimitExceeded{
				TestId:                testId,
				IsCPUTimeExceeded:     flags.TimeLimitExceeded,
				MemoryLimitExceeded:   flags.MemoryLimitExceeded,
				IdlenessLimitExceeded: flags.IdlenessLimitExceeded,
			},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) FinishTestWithRuntimeError(testId int64) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_FinishTestWithRuntimeError{
			FinishTestWithRuntimeError: &msg.FinishTestWithRuntimeError{
				TestId: testId,
			},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) FinishTestWithVerdictAccepted(testId int64) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_FinishTestWithVerdictAccepted{
			FinishTestWithVerdictAccepted: &msg.FinishTestWithVerdictAccepted{
				TestId: testId,
			},
		},
	}
	r.sendEvalResponse(m)
}

func (r *Gatherer) FinishTestWithVerdictWrongAnswer(testId int64) {
	m := &msg.EvaluationFeedback{
		FeedbackTypes: &msg.EvaluationFeedback_FinishTestWithVerdictWrongAnswer{
			FinishTestWithVerdictWrongAnswer: &msg.FinishTestWithVerdictWrongAnswer{
				TestId: testId,
			},
		},
	}
	r.sendEvalResponse(m)
}
