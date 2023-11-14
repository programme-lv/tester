package testing

type RuntimeMetrics struct {
	CpuTimeMillis  int64
	WallTimeMillis int64
	MemoryKBytes   int64
}

type RuntimeOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int64
}

type RuntimeData struct {
	Output  RuntimeOutput
	Metrics RuntimeMetrics
}

type RuntimeExceededFlags struct {
	TimeLimitExceeded     bool
	MemoryLimitExceeded   bool
	IdlenessLimitExceeded bool
}

type EvalResGatherer interface {
	StartEvaluation()
	FinishWithInternalServerError(error)
	FinishEvaluation()

	StartCompilation()
	FinishCompilation(data *RuntimeData)
	FinishWithCompilationError()

	StartTesting(maxScore int64)
	IgnoreTest(testId int64)

	StartTest(testId int64)
	ReportTestSubmissionRuntimeData(testId int64, rd *RuntimeData)

	FinishTestWithLimitExceeded(testId int64, flags RuntimeExceededFlags)
	FinishTestWithRuntimeError(testId int64)

	ReportTestCheckerRuntimeData(testId int64, rd *RuntimeData)

	FinishTestWithVerdictAccepted(testId int64)
	FinishTestWithVerdictWrongAnswer(testId int64)

	IncrementScore(delta int64)
}

// TODO: add reporting tester information somewhere
