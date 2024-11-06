package internal

type RuntimeData struct {
	Stdout   []byte `json:"stdout"`
	Stderr   []byte `json:"stderr"`
	ExitCode int64  `json:"exit_code"`

	CpuMillis     int64 `json:"cpu_time_millis"`
	WallMillis    int64 `json:"wall_time_millis"`
	MemoryKiBytes int64 `json:"memory_kibibytes"`

	CtxSwVoluntary int64 `json:"context_switches_voluntary"`
	CtxSwForced    int64 `json:"context_switches_forced"`

	ExitSignal    *int64 `json:"exit_signal"`
	IsolateStatus string `json:"isolate_status"`
}

type EvalReq struct {
	EvalUuid  string `json:"eval_uuid"`
	ResSqsUrl string `json:"res_sqs_url"`

	Code       string    `json:"code"`
	Language   Language  `json:"language"`
	Tests      []ReqTest `json:"tests"`
	Checker    *string   `json:"checker"`
	Interactor *string   `json:"interactor"`

	CpuMillis int `json:"cpu_millis"`
	MemoryKiB int `json:"memory_kib"`
}

type ReqTest struct {
	ID int `json:"id"`

	InputSha256  string  `json:"input_sha256"`
	InputS3Url   *string `json:"input_s3_url"`
	InputContent *string `json:"input_content"`
	InputHttpUrl *string `json:"input_http_url"`

	AnswerSha256  string  `json:"answer_sha256"`
	AnswerS3Url   *string `json:"answer_s3_url"`
	AnswerContent *string `json:"answer_content"`
	AnswerHttpUrl *string `json:"answer_http_url"`
}

type Language struct {
	LangID        string  `json:"lang_id"`
	LangName      string  `json:"lang_name"`
	CodeFname     string  `json:"code_fname"`
	CompileCmd    *string `json:"compile_cmd"`
	CompiledFname *string `json:"compiled_fname"`
	ExecCmd       string  `json:"exec_cmd"`
}
