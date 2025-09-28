package internal

type RunData struct {
	Stdin    []byte
	Stdout   []byte
	Stderr   []byte
	ExitCode int64

	CpuMs  int64
	WallMs int64
	MemKiB int64

	CtxSwV int64
	CtxSwF int64

	ExitSignal    *int64
	IsolateStatus *string
	IsolateMsg    *string

	FullReport string
}
