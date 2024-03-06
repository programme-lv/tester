package messaging

type Limits struct {
	CPUTimeMillis int `json:"cpu_time_millis"`
	MemKibibytes  int `json:"mem_kibibytes"`
}

// TestRef is used to reference input and answer
// text files stored on DigitalOcean Spaces
// with their respective SHA256 hash values
type TestRef struct {
	ID        int    `json:"id"`
	InSHA256  string `json:"in_sha256"`
	AnsSHA256 string `json:"ans_sha256"`
}

// DOSpacesAuth is used to authenticate with DigitalOcean Spaces
type DOSpacesAuth struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type Subtask struct {
	ID      int   `json:"id"`
	Score   int   `json:"score"`
	TestIDs []int `json:"test_ids"`
}

// PLanguage is used to specify the programming language
type PLanguage struct {
	ID           string `json:"id"`
	FullName     string `json:"full_name"`
	CodeFilename string `json:"code_filename"`
	CompileCmd   string `json:"compile_cmd"`
	ExecCmd      string `json:"exec_cmd"`
}

type EvaluationRequest struct {
	Submission string    `json:"submission"`
	PLanguage  PLanguage `json:"planguage"`
	Limits     Limits    `json:"limits"`
	EvalTypeID string    `json:"eval_type_id"`

	Tests    []TestRef `json:"tests"`
	Subtasks []Subtask `json:"subtasks"`

	TestlibChecker *string `json:"testlib_checker"`

	DOSpacesAuth *DOSpacesAuth `json:"do_spaces_auth"`
}

type ResponseCorrelation struct {
	UnixMillis  int64 `json:"unix_millis"`
	RandomInt63 int64 `json:"random_int_63"`
}
