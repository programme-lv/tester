package tester

import (
	"github.com/programme-lv/tester/internal"
)

type ResultGatherer interface {
	StartJob(systemInfo string)

	StartCompile()
	FinishCompile(data *internal.RuntimeData)

	ReachTest(testId int64, input []byte, answer []byte)
	IgnoreTest(testId int64)
	FinishTest(testId int64,
		submission *internal.RuntimeData,
		checker *internal.RuntimeData)

	CompileError(msg string)
	InternalError(msg string)
	FinishNoError()
}
