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

func SelectProgrammingLanguageById(
	db *sqlx.DB,
	programmingLanguageId string,
) (*ProgrammingLanguage, error) {
	var programmingLanguage *ProgrammingLanguage = &ProgrammingLanguage{}
	err := db.Get(
		programmingLanguage,
		"SELECT id, full_name, code_filename, compile_cmd, execute_cmd, env_version_cmd, hello_world_code FROM programming_languages WHERE id = $1",
		programmingLanguageId,
	)
	return programmingLanguage, err
}
