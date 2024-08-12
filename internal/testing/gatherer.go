package testing

import "github.com/programme-lv/tester/internal"

type EvalResGatherer interface {
	StartEvaluation(systemInfo string)
	FinishEvaluation(errIfAny error)

	StartCompilation()
	FinishCompilation(data *internal.RuntimeData)

	StartTesting()
	FinishTesting()

	IgnoreTest(testId int64)

	StartTest(testId int64)
	FinishTest(testId int64,
		submission *internal.RuntimeData,
		checker *internal.RuntimeData)
}
