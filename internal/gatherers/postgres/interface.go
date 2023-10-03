package postgres

import "github.com/programme-lv/tester/internal/testing"

func (g *Gatherer) StartEvaluation(testerInfo string) {
	//TODO implement me

	panic("implement me")
}

func (g *Gatherer) FinishWithInternalServerError(err error) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) FinishEvaluation() {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) StartCompilation() {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) FinishCompilation(data testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) FinishWithCompilationError(err error) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) StartTesting(maxScore int) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) IgnoreTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) StartTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) ReportTestSubmissionRuntimeData(testId int64, rd testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) FinishTestWithLimitExceeded(testId int64, flags testing.RuntimeExceededFlags) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) FinishTestWithRuntimeError(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) ReportTestCheckerRuntimeData(testId int64, rd testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) FinishTestWithVerdictAccepted(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) FinishTestWithVerdictWrongAnswer(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (g *Gatherer) IncrementScore(delta int) {
	//TODO implement me
	panic("implement me")
}
