package api

// Simple, non-streaming response types for execution results

// TestResult represents the result of a single test case
type TestResult struct {
	TestId int32 `json:"test_id"`

	// Runtime data for submission and for checker/interactor
	Subm *RuntimeData `json:"subm"`
	Chkr *RuntimeData `json:"chkr"`
}

// CompileResult represents compilation outcome
type CompileResult struct {
	Success bool    `json:"success"`
	Error   *string `json:"error"`

	// Resource usage during compilation
	CpuMillis  *int64 `json:"cpu_ms"`
	WallMillis *int64 `json:"wall_ms"`
	RamKiBytes *int64 `json:"ram_kib"`
}

type ExecStatus string

const (
	Success       ExecStatus = "success"
	CompileError  ExecStatus = "compile_error"
	InternalError ExecStatus = "internal_error"
)

// ExecResponse is a simple, complete response for code execution
type ExecResponse struct {
	EvalUuid string `json:"eval_uuid"`

	// Overall execution status
	Status ExecStatus `json:"status"`

	// Compilation result
	Compilation *RuntimeData `json:"compilation"`

	// Test results (empty if compilation failed)
	TestResults []TestResult `json:"test_results"`

	// Overall error message (for internal errors)
	ErrorMsg *string `json:"error_msg"`

	// Execution metadata
	StartTime   string `json:"start_time"`
	FinishTime  string `json:"finish_time"`
	TotalTimeMs int64  `json:"total_time_ms"`

	// System information
	SystemInfo string `json:"system_info"`
}
