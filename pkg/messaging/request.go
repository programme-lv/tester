package messaging

type Limits struct {
	CPUTimeMillis int `json:"cpu_time_millis"`
	MemKibibytes  int `json:"mem_kibibytes"`
}

type TestRef struct {
	ID int `json:"id"`

	InContent  *string `json:"in_content"`
	InSHA256   string  `json:"in_sha256"`
	InDownlUrl *string `json:"in_downl_url"` // maybe pre-signed s3 req url

	AnsContent  *string `json:"ans_content"`
	AnsSHA256   string  `json:"ans_sha256"`
	AnsDownlUrl *string `json:"ans_downl_url"` // maybe pre-signed s3 req url

}

type Subtask struct {
	ID      int   `json:"id"`
	Score   int   `json:"score"`
	TestIDs []int `json:"test_ids"`
}

// PLanguage is used to specify the programming language
type PLanguage struct {
	ID               string  `json:"id"`
	FullName         string  `json:"full_name"`
	CodeFilename     string  `json:"code_filename"`
	CompileCmd       *string `json:"compile_cmd"`
	CompiledFilename *string `json:"compiled_filename"`
	ExecCmd          string  `json:"exec_cmd"`
}

type EvaluationRequest struct {
	Submission string    `json:"submission"`
	PLanguage  PLanguage `json:"planguage"`
	Limits     Limits    `json:"limits"`
	EvalTypeID string    `json:"eval_type_id"`

	Tests    []TestRef `json:"tests"`
	Subtasks []Subtask `json:"subtasks"`

	TestlibChecker string `json:"testlib_checker"`
}

type ResponseCorrelation struct {
	UnixMillis  int64 `json:"unix_millis"`
	RandomInt63 int64 `json:"random_int_63"`
}
