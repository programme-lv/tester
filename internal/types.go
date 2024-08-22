package internal

type RuntimeData struct {
	Stdout   *string `json:"stdout"`
	Stderr   *string `json:"stderr"`
	ExitCode int64   `json:"exit_code"`

	CpuTimeMillis   int64 `json:"cpu_time_millis"`
	WallTimeMillis  int64 `json:"wall_time_millis"`
	MemoryKibiBytes int64 `json:"memory_kibibytes"`

	ContextSwitchesVoluntary int64 `json:"context_switches_voluntary"`
	ContextSwitchesForced    int64 `json:"context_switches_forced"`

	ExitSignal    *int64 `json:"exit_signal"`
	IsolateStatus string `json:"isolate_status"`
}

type EvaluationRequest struct {
	Submission     string              `json:"submission"`
	Language       ProgrammingLanguage `json:"language"`
	Limits         ExecutionLimits     `json:"limits"`
	Tests          []RequestTest       `json:"tests"`
	TestlibChecker string              `json:"testlib_checker"`
}

type RequestTest struct {
	Id int64 `json:"id"`

	InputSha256  string  `json:"input_sha256"`
	InputS3Uri   *string `json:"input_s3_uri"`
	InputContent *string `json:"input_content"`
	InputHttpUrl *string `json:"input_http_url"`

	AnswerSha256  string  `json:"answer_sha256"`
	AnswerS3Uri   *string `json:"answer_s3_uri"`
	AnswerContent *string `json:"answer_content"`
	AnswerHttpUrl *string `json:"answer_http_url"`
}

type ProgrammingLanguage struct {
	Id string `json:"id"`

	LanguageName    string `json:"name"`
	SourceCodeFname string `json:"code_filename"`

	CompileCommand   *string `json:"compile_cmd"`
	CompiledFilename *string `json:"compiled_filename"`

	ExecuteCommand string `json:"exec_cmd"`
}

type ExecutionLimits struct {
	CpuTimeMillis   int64 `json:"cpu_time_millis"`
	MemoryKibiBytes int64 `json:"memory_kibibytes"`
}
