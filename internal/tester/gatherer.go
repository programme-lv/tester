package tester

import (
	pkg "github.com/programme-lv/tester/pkg"
)

type EvalResGatherer interface {
	StartEvaluation(systemInfo string)
	FinishEvalWithCompileError(msg string)
	FinishEvalWithInternalError(msg string)
	FinishEvalWithoutError()

	StartCompilation()
	FinishCompilation(data *pkg.RuntimeData)

	StartTesting()
	FinishTesting()

	ReachTest(testId int64, input []byte, answer []byte)
	IgnoreTest(testId int64)
	FinishTest(testId int64,
		submission *pkg.RuntimeData,
		checker *pkg.RuntimeData)
}
