package messaging

type Submission struct {
	SourceCode string
	LanguageId string
}

type EvaluationRequest struct {
	TaskVersionId int
	Submission    Submission
}

type Correlation struct {
	IsEvaluation bool  `json:"is_evaluation"`
	EvaluationId int64 `json:"evaluation_id,omitempty"`
	UnixMillis   int64 `json:"unix_millis"`
	RandomInt63  int64 `json:"random_int_63"`
}

type RuntimeOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

type RuntimeMetrics struct {
	CpuTimeMillis  float64 `json:"cpu_time_millis"`
	WallTimeMillis float64 `json:"wall_time_millis"`
	MemoryKBytes   int     `json:"memory_k_bytes"`
}

type RuntimeExceededFlags struct {
	TimeLimitExceeded     bool `json:"time_limit_exceeded"`
	MemoryLimitExceeded   bool `json:"memory_limit_exceeded"`
	IdlenessLimitExceeded bool `json:"idleness_limit_exceeded"`
}

type RuntimeData struct {
	Output  RuntimeOutput
	Metrics RuntimeMetrics
}

type EvaluationResponse struct {
}
