package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/tester/internal/database"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/pkg/messaging/statuses"
)

type Gatherer struct {
	postgres         *sqlx.DB
	submissionEvalId int64
}

func NewPostgresGatherer(postgres *sqlx.DB, submissionEvalId int64) *Gatherer {
	return &Gatherer{
		postgres:         postgres,
		submissionEvalId: submissionEvalId,
	}
}

func (g *Gatherer) StartEvaluation() {
	err := database.UpdateSubmissionEvaluationEvalStatusId(
		g.postgres,
		statuses.Received,
		g.submissionEvalId,
	)
	panicOnError(err)
}

func (g *Gatherer) FinishWithInternalServerError(err error) {
	err2 := database.UpdateSubmissionEvaluationEvalStatusId(
		g.postgres,
		statuses.InternalServerError,
		g.submissionEvalId,
	)
	panicOnError(err)
	panicOnError(err2)
}

func (g *Gatherer) FinishEvaluation() {
	err := database.UpdateSubmissionEvaluationEvalStatusId(
		g.postgres,
		statuses.Finished,
		g.submissionEvalId,
	)
	panicOnError(err)
}

func (g *Gatherer) StartCompilation() {
	err := database.UpdateSubmissionEvaluationEvalStatusId(
		g.postgres,
		statuses.Compiling,
		g.submissionEvalId,
	)
	panicOnError(err)
}

func (g *Gatherer) FinishCompilation(data *testing.RuntimeData) {
	//TODO implement me
	// update compilation_stdout, compilation_stderr, and so on
	panic("implement me")
}

func (g *Gatherer) FinishWithCompilationError() {
	//TODO implement me
	// update eval_status_id to CompilationError
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

var _ testing.EvalResGatherer = (*Gatherer)(nil)

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
