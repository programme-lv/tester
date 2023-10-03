package testing

type RuntimeMetrics struct {
	CpuTimeMillis  float64
	WallTimeMillis float64
	MemoryKBytes   int
}

type RuntimeOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
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

	StartTesting(maxScore int)
	IgnoreTest(testId int64)

	StartTest(testId int64)
	ReportTestSubmissionRuntimeData(testId int64, rd RuntimeData)

	FinishTestWithLimitExceeded(testId int64, flags RuntimeExceededFlags)
	FinishTestWithRuntimeError(testId int64)

	ReportTestCheckerRuntimeData(testId int64, rd RuntimeData)

	FinishTestWithVerdictAccepted(testId int64)
	FinishTestWithVerdictWrongAnswer(testId int64)

	IncrementScore(delta int)
}

// TODO: add reporting tester information somewhere
