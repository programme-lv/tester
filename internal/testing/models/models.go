package models

type ExecutableFile struct {
	Content []byte
	ExecCmd string
}

type TestPaths struct {
	ID         int
	InputPath  string
	AnswerPath string
}

type Subtask struct {
	ID      int
	Score   int
	TestIDs []int
}

type Constraints struct {
	CpuTimeLimInSec float64
	MemoryLimitInKB int64
}

type ArrangedEvaluationReq struct {
	Submission  ExecutableFile
	SubmConstrs Constraints
	Checker     ExecutableFile

	Tests    []TestPaths
	Subtasks []Subtask
}

type RuntimeMetrics struct {
	CpuTimeMillis  int64
	WallTimeMillis int64
	MemoryKBytes   int64
}

type RuntimeOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int64
}

type RuntimeData struct {
	Output  RuntimeOutput
	Metrics RuntimeMetrics
}

type RuntimeExceededFlags struct {
	TimeLimitExceeded     bool
	MemoryLimitExceeded   bool
	IdlenessLimitExceeded bool
}
