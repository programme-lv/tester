package testing

type CompiledFile struct {
	Content  []byte
	Filename string
	ExecCmd  string
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
	Submission  CompiledFile
	SubmConstrs Constraints
	Checker     CompiledFile

	Tests    []TestPaths
	Subtasks []Subtask
}
