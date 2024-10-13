package sqsgath

import (
	"strings"

	"github.com/programme-lv/tester/internal"
)

func trimRuntimeData(data *internal.RuntimeData, maxHeight int, maxWidth int) *internal.RuntimeData {
	if data == nil {
		return nil
	}

	return &internal.RuntimeData{
		Stdout:                   trimStringToRectangle(data.Stdout, maxHeight, maxWidth),
		Stderr:                   trimStringToRectangle(data.Stderr, maxHeight, maxWidth),
		ExitCode:                 data.ExitCode,
		CpuTimeMillis:            data.CpuTimeMillis,
		WallTimeMillis:           data.WallTimeMillis,
		MemoryKibiBytes:          data.MemoryKibiBytes,
		ContextSwitchesVoluntary: data.ContextSwitchesVoluntary,
		ContextSwitchesForced:    data.ContextSwitchesForced,
		ExitSignal:               data.ExitSignal,
		IsolateStatus:            data.IsolateStatus,
	}
}

func trimStringToRectangle(s *string, maxHeight int, maxWidth int) *string {
	if s == nil {
		return nil
	}
	// split into lines
	res := ""
	lines := strings.Split(*s, "\n")
	if len(lines) > maxHeight {
		lines = lines[:maxHeight]
		lines = append(lines, "[...]")
	}
	for i, line := range lines {
		if i > 0 {
			res += "\n"
		}
		if len(line) > maxWidth {
			res += line[:maxWidth] + "[...]"
		} else {
			res += line
		}
	}
	return &res
}
