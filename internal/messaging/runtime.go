package messaging

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

type RuntimeData struct {
	Output  RuntimeOutput
	Metrics RuntimeMetrics
}
