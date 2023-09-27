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
	FinishEvaluation()
	// compilation
	StartCompilation()
	FinishCompilation(RuntimeData)
	// testing
	StartTesting()
	FinishSingleTest(
		testId int64,
		status string,
		submission RuntimeData,
		checker RuntimeData,
	)
	UpdateScore()
}
