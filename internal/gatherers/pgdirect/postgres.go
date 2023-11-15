package pgdirect

import (
	"github.com/go-jet/jet/v2/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/tester/internal/database/proglv/public/model"
	"github.com/programme-lv/tester/internal/database/proglv/public/table"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/pkg/messaging/statuses"
	"log"
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
	err := table.RuntimeData.INSERT(
		table.RuntimeData.Stdout,
		table.RuntimeData.Stderr,
		table.RuntimeData.TimeMillis,
		table.RuntimeData.MemoryKibibytes,
		table.RuntimeData.TimeWallMillis,
		table.RuntimeData.ExitCode,
	).
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
	stmt := table.Evaluations.UPDATE(table.Evaluations.EvalStatusID).
		SET(statuses.CompilationError).
		WHERE(table.Evaluations.ID.EQ(postgres.Int64(g.evaluationId)))

	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) StartTesting(maxScore int64) {
	stmt := table.Evaluations.UPDATE(table.Evaluations.EvalStatusID, table.Evaluations.EvalPossibleScore).
		SET(statuses.Testing, maxScore).
		WHERE(table.Evaluations.ID.EQ(postgres.Int64(g.evaluationId)))
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) IgnoreTest(testId int64) {
	stmt := table.EvaluationTestResults.INSERT(
		table.EvaluationTestResults.EvaluationID,
		table.EvaluationTestResults.EvalStatusID,
		table.EvaluationTestResults.TaskVTestID,
	).VALUES(
		g.evaluationId,
		statuses.Ignored,
		testId,
	)
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) StartTest(testId int64) {
	stmt := table.EvaluationTestResults.INSERT(
		table.EvaluationTestResults.EvaluationID,
		table.EvaluationTestResults.EvalStatusID,
		table.EvaluationTestResults.TaskVTestID,
	).VALUES(
		g.evaluationId,
		statuses.Testing,
		testId,
	)
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) ReportTestSubmissionRuntimeData(testId int64, rd *testing.RuntimeData) {
	mrd := model.RuntimeData{
		Stdout:          &rd.Output.Stdout,
		Stderr:          &rd.Output.Stderr,
		TimeMillis:      &rd.Metrics.CpuTimeMillis,
		MemoryKibibytes: &rd.Metrics.MemoryKBytes,
		TimeWallMillis:  &rd.Metrics.WallTimeMillis,
		ExitCode:        &rd.Output.ExitCode,
	}
	stmt := table.RuntimeData.INSERT(table.RuntimeData.Stdout,
		table.RuntimeData.Stderr,
		table.RuntimeData.TimeMillis,
		table.RuntimeData.MemoryKibibytes,
		table.RuntimeData.TimeWallMillis,
		table.RuntimeData.ExitCode,
	).
		MODEL(&mrd).
		RETURNING(table.RuntimeData.ID)
	err := stmt.Query(g.postgres, &mrd)
	panicOnError(err)

	stmt2 := table.EvaluationTestResults.UPDATE(table.EvaluationTestResults.ExecRDataID).
		SET(mrd.ID).
		WHERE(table.EvaluationTestResults.EvaluationID.EQ(postgres.Int64(g.evaluationId)).AND(
			table.EvaluationTestResults.TaskVTestID.EQ(postgres.Int64(testId))))
	_, err = stmt2.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) FinishTestWithLimitExceeded(testId int64, flags testing.RuntimeExceededFlags) {
	status := statuses.IdlenessLimitExceeded
	if flags.MemoryLimitExceeded {
		status = statuses.MemoryLimitExceeded
	}
	if flags.TimeLimitExceeded {
		status = statuses.TimeLimitExceeded
	}

	stmt := table.EvaluationTestResults.UPDATE(table.EvaluationTestResults.EvalStatusID).
		SET(status).
		WHERE(table.EvaluationTestResults.EvaluationID.EQ(postgres.Int64(g.evaluationId)).AND(
			table.EvaluationTestResults.TaskVTestID.EQ(postgres.Int64(testId))))
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) FinishTestWithRuntimeError(testId int64) {
	stmt := table.EvaluationTestResults.UPDATE(table.EvaluationTestResults.EvalStatusID).
		SET(statuses.RuntimeError).
		WHERE(table.EvaluationTestResults.EvaluationID.EQ(postgres.Int64(g.evaluationId)).AND(
			table.EvaluationTestResults.TaskVTestID.EQ(postgres.Int64(testId))))
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) ReportTestCheckerRuntimeData(testId int64, rd *testing.RuntimeData) {
	mrd := model.RuntimeData{
		Stdout:          &rd.Output.Stdout,
		Stderr:          &rd.Output.Stderr,
		TimeMillis:      &rd.Metrics.CpuTimeMillis,
		MemoryKibibytes: &rd.Metrics.MemoryKBytes,
		TimeWallMillis:  &rd.Metrics.WallTimeMillis,
		ExitCode:        &rd.Output.ExitCode,
	}
	stmt := table.RuntimeData.INSERT(
		table.RuntimeData.Stdout,
		table.RuntimeData.Stderr,
		table.RuntimeData.TimeMillis,
		table.RuntimeData.MemoryKibibytes,
		table.RuntimeData.TimeWallMillis,
		table.RuntimeData.ExitCode).
		MODEL(&mrd).
		RETURNING(table.RuntimeData.ID)
	err := stmt.Query(g.postgres, &mrd)
	panicOnError(err)

	stmt2 := table.EvaluationTestResults.UPDATE(table.EvaluationTestResults.CheckerRDataID).
		SET(mrd.ID).
		WHERE(table.EvaluationTestResults.EvaluationID.EQ(postgres.Int64(g.evaluationId)).AND(
			table.EvaluationTestResults.TaskVTestID.EQ(postgres.Int64(testId))))
	_, err = stmt2.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) FinishTestWithVerdictAccepted(testId int64) {
	stmt := table.EvaluationTestResults.UPDATE(table.EvaluationTestResults.EvalStatusID).
		SET(statuses.Accepted).
		WHERE(table.EvaluationTestResults.EvaluationID.EQ(postgres.Int64(g.evaluationId)).AND(
			table.EvaluationTestResults.TaskVTestID.EQ(postgres.Int64(testId))))
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) FinishTestWithVerdictWrongAnswer(testId int64) {
	stmt := table.EvaluationTestResults.UPDATE(table.EvaluationTestResults.EvalStatusID).
		SET(statuses.WrongAnswer).
		WHERE(table.EvaluationTestResults.EvaluationID.EQ(postgres.Int64(g.evaluationId)).AND(
			table.EvaluationTestResults.TaskVTestID.EQ(postgres.Int64(testId))))
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

func (g *Gatherer) IncrementScore(delta int64) {
	stmt := table.Evaluations.UPDATE(table.Evaluations.EvalTotalScore).
		SET(table.Evaluations.EvalTotalScore.ADD(postgres.Int(delta))).
		WHERE(table.Evaluations.ID.EQ(postgres.Int64(g.evaluationId)))
	log.Println(stmt.Sql())
	_, err := stmt.Exec(g.postgres)
	panicOnError(err)
}

var _ testing.EvalResGatherer = (*Gatherer)(nil)

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
