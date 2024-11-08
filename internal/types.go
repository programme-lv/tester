package internal

type RuntimeData struct {
	Stdin    []byte
	Stdout   []byte
	Stderr   []byte
	ExitCode int64

	CpuMillis     int64
	WallMillis    int64
	MemoryKiBytes int64

	CtxSwVoluntary int64
	CtxSwForced    int64

	ExitSignal    *int64
	IsolateStatus string
}
