package messaging

type Submission struct {
	Body   string `json:"body"`
	LangID string `json:"lang_id"`
}

type Limits struct {
	CPUTimeMillis int `json:"cpu_time_millis"`
	MemKibibytes  int `json:"mem_kibibytes"`
}

type Test struct {
	ID        int    `json:"id"`
	InSHA256  string `json:"in_sha256"`
	AnsSHA256 string `json:"ans_sha256"`
}

type TestResolver struct {
	Type      string `json:"type"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type Subtask struct {
	ID      int   `json:"id"`
	Score   int   `json:"score"`
	TestIDs []int `json:"test_ids"`
}

type EvaluationRequest struct {
	Submission     Submission   `json:"submission"`
	Limits         Limits       `json:"limits"`
	EvalTypeID     string       `json:"eval_type_id"`
	Tests          []Test       `json:"tests"`
	TestResolver   TestResolver `json:"test_resolver"`
	Subtasks       []Subtask    `json:"subtasks"`
	TestlibChecker *string      `json:"testlib_checker"`
}

type ResponseCorrelation struct {
	UnixMillis  int64 `json:"unix_millis"`
	RandomInt63 int64 `json:"random_int_63"`
}
