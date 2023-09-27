package testing

type RuntimeMetrics struct {
	CpuTimeMillis  float64
	WallTimeMillis float64
	MemoryKBytes   int
	ExitCode       int
}

type RuntimeOutput struct {
	Stdout string
	Stderr string
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
	FinishEvaluation()
	// compilation
	StartCompilation()
	FinishCompilation(RuntimeData)
	// testing
	StartTesting()
	StartTestingSingleTest(testId int64)
	FinishTestingSingleTest(
		testId int64,
		submission RuntimeData,
		checker RuntimeData,
	)
	IgnoreTest(testId int64)
}

/*
- AC, PT ( accepted, partial )
- WA, PE ( wrong answer, presentation error )
- TLE, MLE, ILE ( ? limit exceeded )
- RE ( runtime error )
*/
