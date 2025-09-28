package internal

type ResultGatherer interface {
	StartJob(systemInfo string)

	StartCompile()
	FinishCompile(data *RunData)

	ReachTest(testId int64, input []byte, answer []byte)
	IgnoreTest(testId int64)
	FinishTest(testId int64, subm *RunData, chkr *RunData)

	CompileError(msg string)
	InternalError(msg string)
	FinishNoError()
}
