package pgdirect

import (
	"github.com/go-jet/jet/v2/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/tester/internal/database/proglv/public/model"
	"github.com/programme-lv/tester/internal/database/proglv/public/table"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/pkg/messaging/statuses"
	"log/slog"
)

type Gatherer struct {
	postgres      *sqlx.DB
	evaluationId  int64
	evalRandInt63 int64 // TODO: add column to submission_evaluations table & utilize it
}

func NewPostgresGatherer(postgres *sqlx.DB, evaluationId int64, evalRandInt63 int64) *Gatherer {
	return &Gatherer{
		postgres:      postgres,
		evaluationId:  evaluationId,
		evalRandInt63: evalRandInt63,
	}
}

func (g *Gatherer) StartEvaluation() {
	stmt := table.Evaluations.UPDATE(table.Evaluations.EvalStatusID).
		SET(statuses.Received).
		WHERE(table.Evaluations.ID.EQ(postgres.Int64(g.evaluationId)))

	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) FinishWithInternalServerError(testingErr error) {
	slog.Error(testingErr.Error())
	stmt := table.Evaluations.UPDATE(table.Evaluations.EvalStatusID).
		SET(statuses.InternalServerError).
		WHERE(table.Evaluations.ID.EQ(postgres.Int64(g.evaluationId)))
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) FinishEvaluation() {
	stmt := table.Evaluations.UPDATE(table.Evaluations.EvalStatusID).
		SET(statuses.Finished).
		WHERE(table.Evaluations.ID.EQ(postgres.Int64(g.evaluationId)))
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) StartCompilation() {
	stmt := table.Evaluations.UPDATE(table.Evaluations.EvalStatusID).
		SET(statuses.Compiling).
		WHERE(table.Evaluations.ID.EQ(postgres.Int64(g.evaluationId)))
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) FinishCompilation(data *testing.RuntimeData) {
	runtimeData := &model.RuntimeData{
		Stdout:          &data.Output.Stdout,
		Stderr:          &data.Output.Stderr,
		TimeMillis:      &data.Metrics.CpuTimeMillis,
		MemoryKibibytes: &data.Metrics.MemoryKBytes,
		TimeWallMillis:  &data.Metrics.WallTimeMillis,
		ExitCode:        &data.Output.ExitCode,
	}
	err := table.RuntimeData.INSERT(table.RuntimeData.AllColumns).
		MODEL(runtimeData).
		RETURNING(table.RuntimeData.ID).
		Query(g.postgres, runtimeData)
	panicOnError(err)

	stmt := table.Evaluations.UPDATE(table.Evaluations.CompilationDataID).
		SET(runtimeData.ID).
		WHERE(table.Evaluations.ID.EQ(postgres.Int64(g.evaluationId)))

	_, err = stmt.Exec(g.postgres)
	panicOnError(err)
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

func (g *Gatherer) ReportTestSubmissionRuntimeData(testId int64, rd *testing.RuntimeData) {
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

func (g *Gatherer) ReportTestCheckerRuntimeData(testId int64, rd *testing.RuntimeData) {
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
