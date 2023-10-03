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
	TimeLimitExceeded        bool
	MemoryLimitExceeded   bool
	IdlenessLimitExceeded bool
}

type EvalResGatherer interface {
	StartEvaluation(
		testerInfo string,
		evalMaxScore int,
	)
	IncrementScore(delta int)
	FinishWithInternalServerError(error)
	FinishEvaluation()

	StartCompilation()
	FinishCompilation(RuntimeData)

	StartTesting()
	IgnoreTest(testId int64)

	StartTest(testId int64)
	ReportTestUserRuntimeData(testId int64, rd RuntimeData)
	ReportTestUserRuntimeLimitExceeded(testId int64, flags RuntimeExceededFlags)
	ReportTestUserRuntimeError(testId int64)

	ReportTestCheckerRuntimeData(testId int64, rd RuntimeData)

	ReportTestVerdictAccepted(testId int64)
	ReportTestVerdictWrongAnswer(testId int64)

	FinishTest(testId int64)
}
