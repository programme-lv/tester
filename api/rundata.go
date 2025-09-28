package api

// RuntimeData contains execution information for a process (streaming version)
type RuntimeData struct {
	Stdin    string `json:"in"`
	Stdout   string `json:"out"`
	Stderr   string `json:"err"`
	ExitCode int64  `json:"exit"`

	CpuMillis  int64 `json:"cpu_ms"`
	WallMillis int64 `json:"wall_ms"`
	RamKiBytes int64 `json:"ram_kib"`

	CtxSwV int64 `json:"ctx_sw_v"`
	CtxSwF int64 `json:"ctx_sw_f"`

	ExitSignal  *int64 `json:"signal"`
	CgOomKilled bool   `json:"cg_oom_killed"` // killed on memory allocation?

	IsolateStatus *string `json:"isolate_status"`
	IsolateMsg    *string `json:"isolate_msg"`
}
