package messaging

type RuntimeOutput struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int64  `json:"exit_code"`
}

type RuntimeMetrics struct {
	CpuTimeMillis  int64 `json:"cpu_time_millis"`
	WallTimeMillis int64 `json:"wall_time_millis"`
	MemoryKBytes   int64 `json:"memory_k_bytes"`
}

type RuntimeData struct {
	Output  RuntimeOutput
	Metrics RuntimeMetrics
}
