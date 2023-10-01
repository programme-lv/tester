package gathering

import "github.com/programme-lv/tester/internal/testing"

type RabbitMQGatherer struct {
	correlationIsEvaluation bool
}

func (r RabbitMQGatherer) StartEvaluation(testerInfo string, evalMaxScore int) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) IncrementScore(delta int) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) FinishWithInternalServerError(err error) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) FinishEvaluation() {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) StartCompilation() {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) FinishCompilation(data testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) StartTesting() {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) IgnoreTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) StartTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestUserRuntimeData(testId int64, rd testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestUserRuntimeLimitExceeded(testId int64, flags testing.RuntimeExceededFlags) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestUserRuntimeError(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestCheckerRuntimeData(testId int64, rd testing.RuntimeData) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestVerdictAccepted(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) ReportTestVerdictWrongAnswer(testId int64) {
	//TODO implement me
	panic("implement me")
}

func (r RabbitMQGatherer) FinishTest(testId int64) {
	//TODO implement me
	panic("implement me")
}

var _ testing.EvalResGatherer = (*RabbitMQGatherer)(nil)
