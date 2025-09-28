package internal

import "github.com/programme-lv/tester/api"

type ResultGatherer interface {
	StartJob(systemInfo string)

	StartCompile()
	FinishCompile(data *api.RuntimeData)

	ReachTest(testId int64, input []byte, answer []byte)
	IgnoreTest(testId int64)
	FinishTest(testId int64, subm *api.RuntimeData, chkr *api.RuntimeData)

	CompileError(msg string)
	InternalError(msg string)
	FinishNoError()
}
