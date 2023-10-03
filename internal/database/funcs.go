package database

import (
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/tester/internal/messaging/statuses"
)

func UpdateSubmissionEvaluationEvalStatusId(
	db sqlx.Execer,
	evalStatusId statuses.Status,
	submissionEvaluationId int64,
) error {
	_, err := db.Exec(
		"UPDATE submission_evaluations SET eval_status_id = $1 WHERE id = $2",
		evalStatusId,
		submissionEvaluationId,
	)
	return err
}
