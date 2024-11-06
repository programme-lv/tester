package sqsgath

import (
	"strings"

	"github.com/programme-lv/tester/internal"
)

func trimRunDataOutput(data *internal.RuntimeData, maxHeight int, maxWidth int) *internal.RuntimeData {
	if data == nil {
		return nil
	}

	return &internal.RuntimeData{
		Stdout:         trimStrToRect(data.Stdout, maxHeight, maxWidth),
		Stderr:         trimStrToRect(data.Stderr, maxHeight, maxWidth),
		ExitCode:       data.ExitCode,
		CpuMillis:      data.CpuMillis,
		WallMillis:     data.WallMillis,
		MemoryKiBytes:  data.MemoryKiBytes,
		CtxSwVoluntary: data.CtxSwVoluntary,
		CtxSwForced:    data.CtxSwForced,
		ExitSignal:     data.ExitSignal,
		IsolateStatus:  data.IsolateStatus,
	}
}

func trimStrToRect(s []byte, maxHeight int, maxWidth int) []byte {
	if s == nil {
		return nil
	}
	// split into lines
	res := ""
	lines := strings.Split(string(s), "\n")
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
	return []byte(res)
}
