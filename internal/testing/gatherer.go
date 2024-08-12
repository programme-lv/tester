package testing

type EvalResGatherer interface {
	StartEvaluation(systemInfo string)
	FinishEvaluation(errIfAny error)

	StartCompilation()
	FinishCompilation(data *RuntimeData)

	StartTesting()
	FinishTesting()

	IgnoreTest(testId int64)

	StartTest(testId int64)
	FinishTest(testId int64,
		submission *RuntimeData,
		checker *RuntimeData)
}
