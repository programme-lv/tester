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
		"SELECT id, full_name, code_filename, compile_cmd, execute_cmd, env_version_cmd, hello_world_code, compiled_filename FROM programming_languages WHERE id = $1",
		programmingLanguageId,
	)
	return programmingLanguage, err
}

func SelectTaskVersionTestsByTaskVersionId(
	db *sqlx.DB,
	taskVersionId int,
) ([]TaskVersionTest, error) {
	var taskVersionTests []TaskVersionTest
	err := db.Select(
		&taskVersionTests,
		"SELECT id, test_filename, task_version_id, input_text_file_id, answer_text_file_id FROM task_version_tests WHERE task_version_id = $1",
		taskVersionId,
	)
	return taskVersionTests, err
}

func SelectTextFileById(
	db *sqlx.DB,
	textFileId int64,
) (*TextFile, error) {
	var textFile TextFile
	err := db.Get(
		&textFile,
		"SELECT * FROM text_files WHERE id = $1",
		textFileId,
	)
	return &textFile, err
}
