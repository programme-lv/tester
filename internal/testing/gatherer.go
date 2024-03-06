package testing

import "github.com/programme-lv/tester/internal/testing/models"

type EvalResGatherer interface {
	StartEvaluation()
	FinishWithInternalServerError(error)
	FinishEvaluation()

	StartCompilation()
	FinishCompilation(data *models.RuntimeData)
	FinishWithCompilationError()

	StartTesting(maxScore int64)
	IgnoreTest(testId int64)

	StartTest(testId int64)
	ReportTestSubmissionRuntimeData(testId int64, rd *models.RuntimeData)

	FinishTestWithLimitExceeded(testId int64, flags models.RuntimeExceededFlags)
	FinishTestWithRuntimeError(testId int64)

	ReportTestCheckerRuntimeData(testId int64, rd *models.RuntimeData)

	FinishTestWithVerdictAccepted(testId int64)
	FinishTestWithVerdictWrongAnswer(testId int64)

	IncrementScore(delta int64)
}

// TODO: add reporting tester information somewhere
