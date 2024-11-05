package utils

import (
	"fmt"
	"io"

	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
)

func CollectProcessRuntimeData(process *isolate.Process) (*internal.RuntimeData, error) {
	err := process.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}

	stderr, err := io.ReadAll(process.Stderr())
	if err != nil {
		return nil, fmt.Errorf("failed to read stderr: %w", err)
	}

	metrics, err := process.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for process: %w", err)
	}

	return &internal.RuntimeData{
		Stdout:                   stringPtr(string(stdout)),
		Stderr:                   stringPtr(string(stderr)),
		ExitCode:                 metrics.ExitCode,
		CpuTimeMillis:            int64(metrics.TimeSec * 1000),
		WallTimeMillis:           int64(metrics.TimeWallSec * 1000),
		MemoryKibiBytes:          metrics.CgMemKb,
		ContextSwitchesVoluntary: metrics.CswVoluntary,
		ContextSwitchesForced:    metrics.CswForced,
		ExitSignal:               metrics.ExitSig,
		IsolateStatus:            metrics.Status,
	}, nil
}

func stringPtr(s string) *string {
	return &s
}
