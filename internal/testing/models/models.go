package models

type ExecutableFile struct {
	Content  []byte
	Filename string
	ExecCmd  string
}

type Test struct {
	ID        int
	InputSHA  string
	AnswerSHA string
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

type PreparedEvaluationReq struct {
	Submission  ExecutableFile
	SubmConstrs Constraints
	Checker     ExecutableFile

	Tests    []Test
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
