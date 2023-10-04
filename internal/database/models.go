package database

import "time"

type TaskVersion struct {
	ID            int64      `db:"id"`
	TaskID        int64      `db:"task_id"`
	ShortCode     string     `db:"short_code"`
	FullName      string     `db:"full_name"`
	TimeLimMs     int        `db:"time_lim_ms"`
	MemLimKb      int        `db:"mem_lim_kb"`
	TestingTypeID string     `db:"testing_type_id"`
	Origin        *string    `db:"origin"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     *time.Time `db:"updated_at"`
	CheckerID     *int64     `db:"checker_id"`
	InteractorID  *int64     `db:"interactor_id"`
}

type ProgrammingLanguage struct {
	ID             string  `db:"id"`
	FullName       string  `db:"full_name"`
	CodeFilename   string  `db:"code_filename"`
	CompileCmd     *string `db:"compile_cmd"`
	ExecuteCmd     string  `db:"execute_cmd"`
	EnvVersionCmd  string  `db:"env_version_cmd"`
	HelloWorldCode string  `db:"hello_world_code"`
}

type SubmissionEvaluation struct {
	ID                  int64     `db:"id"`
	TaskSubmissionID    int64     `db:"task_submission_id"`
	EvalTaskVersionID   int64     `db:"eval_task_version_id"`
	TestMaximumTimeMs   *int64    `db:"test_maximum_time_ms"`
	TestMaximumMemoryKb *int64    `db:"test_maximum_memory_kb"`
	TestTotalTimeMs     int64     `db:"test_total_time_ms"`
	TestTotalMemoryKb   int64     `db:"test_total_memory_kb"`
	EvalStatusId        string    `db:"eval_status_id"`
	EvalTotalScore      int64     `db:"eval_total_score"`
	EvalPossibleScore   *int64    `db:"eval_possible_score"`
	CompilationStdout   *string   `db:"compilation_stdout"`
	CompilationStderr   *string   `db:"compilation_stderr"`
	CompilationTimeMs   *int64    `db:"compilation_time_ms"`
	CompilationMemoryKb *int64    `db:"compilation_memory_kb"`
	CreatedAt           time.Time `db:"created_at"`
	UpdateAt            time.Time `db:"updated_at"`
}

type TaskVersionTest struct {
	ID               int64  `db:"id"`
	TestFilename     string `db:"test_filename"`
	TaskVersionID    int64  `db:"task_version_id"`
	InputTextFileID  *int64 `db:"input_text_file_id"`
	AnswerTextFileID *int64 `db:"answer_text_file_id"`
}

type TextFile struct {
	ID        int64     `db:"id"`
	Sha256    string    `db:"sha256"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
}
