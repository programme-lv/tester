package testing

import (
	"io"

	"github.com/programme-lv/tester/internal/isolate"
)

func collectProcessRuntimeData(process *isolate.Process) (*RuntimeData, error) {
	err := process.Start()
	if err != nil {
		return nil, err
	}

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		return nil, err
	}

	stderr, err := io.ReadAll(process.Stderr())
	if err != nil {
		return nil, err
	}

	metrics, err := process.Wait()
	if err != nil {
		return nil, err
	}

	return &RuntimeData{
		Output: RuntimeOutput{
			Stdout:   string(stdout),
			Stderr:   string(stderr),
			ExitCode: metrics.ExitCode,
		},
		Metrics: RuntimeMetrics{
			CpuTimeMillis:  int64(metrics.TimeSec * 1000),
			WallTimeMillis: int64(metrics.TimeWallSec * 1000),
			MemoryKBytes:   metrics.CgMemKb,
		},
	}, nil
}
