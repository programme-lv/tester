package natsgath

import (
	"strings"
)

func trimStrToRect(s string, maxHeight int, maxWidth int) string {
	if s == "" {
		return ""
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
	return res
}
