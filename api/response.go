package api

// Simple, non-streaming response types for execution results

// TestResult represents the result of a single test case
type TestResult struct {
	TestId int32 `json:"test_id"`

	// Resource usage (only if execution succeeded)
	CpuMillis     *int64 `json:"cpu_ms,omitempty"`
	WallMillis    *int64 `json:"wall_ms,omitempty"`
	MemoryKiBytes *int64 `json:"mem_kib,omitempty"`

	// Exit information
	ExitCode   *int64 `json:"exit_code,omitempty"`
	ExitSignal *int64 `json:"exit_signal,omitempty"`

	// Error message if execution failed
	ErrorMessage *string `json:"error_message,omitempty"`

	// Output (truncated for simple response)
	Stdout *string `json:"stdout,omitempty"`
	Stderr *string `json:"stderr,omitempty"`
}

// CompileResult represents compilation outcome
type CompileResult struct {
	Success bool    `json:"success"`
	Error   *string `json:"error,omitempty"`

	// Resource usage during compilation
	CpuMillis     *int64 `json:"cpu_ms,omitempty"`
	WallMillis    *int64 `json:"wall_ms,omitempty"`
	MemoryKiBytes *int64 `json:"mem_kib,omitempty"`
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
	Compilation CompileResult `json:"compilation"`

	// Test results (empty if compilation failed)
	TestResults []TestResult `json:"test_results"`

	// Overall error message (for internal errors)
	ErrorMessage *string `json:"error_message,omitempty"`

	// Execution metadata
	StartTime   string `json:"start_time"`
	FinishTime  string `json:"finish_time"`
	TotalTimeMs int64  `json:"total_time_ms"`

	// System information
	SystemInfo *string `json:"system_info,omitempty"`
}
