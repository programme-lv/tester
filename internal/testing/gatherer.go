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

type EvalResGatherer interface {
	// evaluation
	StartEvaluation(
		testerInfo string,
		evalMaxScore int,
	)
	IncrementScore(delta int)
	FinishWithInernalServerError(error)
	FinishEvaluation() // to be called after all tests are marked as finished

	// compilatihlkkjn
	StartCompilation()
	FinishCompilation(RuntimeData)

	// testing
	StartTesting()
	IgnoreTest(testId int64)

	StartTest(testId int64)
	// runtime data
	ReportTestSubmissionRuntimeData(int64, RuntimeData)
	ReportTestCheckerRuntimeData(int64, RuntimeData)
	// verdicts
	ReportTestVerdictLimitExceeded(testId int64, tle bool, mle bool, ile bool)
	ReportTestRuntimeError(testId int64)
	ReportTestAccepted(testId int64)
	ReportTestWrongAnswer(testId int64)

	FinishTest(testId int64) // to be called after all runtime data and verdicts are reported
}

/*
CHECKER DETERMINED RESULTS
- AC, PT ( accepted, partial )
- WA, PE ( wrong answer, presentation error )
SOLUTION DETERMINED RESULTS
- TLE, MLE, ILE ( ? limit exceeded )
- RE ( runtime error )
*/
