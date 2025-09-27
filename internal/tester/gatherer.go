package tester

import (
	"github.com/programme-lv/tester/internal"
)

type ResultGatherer interface {
	StartEvaluation(systemInfo string)
	FinishEvalWithCompileError(msg string)
	FinishEvalWithInternalError(msg string)
	FinishEvalWithoutError()

	StartCompilation()
	FinishCompilation(data *internal.RuntimeData)

	ReachTest(testId int64, input []byte, answer []byte)
	IgnoreTest(testId int64)
	FinishTest(testId int64,
		submission *internal.RuntimeData,
		checker *internal.RuntimeData)
}
