package tester

import "github.com/programme-lv/tester/api"

const wallLimitMs = 14000

// groupSkip tracks scoring units so remaining tests in a failed unit can
// be ignored. A test is ignored only when every group that contains it
// already has a failed test. Tests that belong to no group always run.
type groupSkip struct {
	byTest map[int][]int
	dead   map[int]bool
}

func newGroupSkip(groups [][]int) *groupSkip {
	s := &groupSkip{
		byTest: make(map[int][]int),
		dead:   make(map[int]bool),
	}
	for gi, tests := range groups {
		for _, id := range tests {
			s.byTest[id] = append(s.byTest[id], gi)
		}
	}
	return s
}

func (s *groupSkip) shouldIgnore(testID int) bool {
	gs := s.byTest[testID]
	if len(gs) == 0 {
		return false
	}
	for _, g := range gs {
		if !s.dead[g] {
			return false
		}
	}
	return true
}

func (s *groupSkip) markFailed(testID int) {
	for _, g := range s.byTest[testID] {
		s.dead[g] = true
	}
}

func testFailed(subm, chkr *api.RuntimeData, cpuMs, ramKiB int32) bool {
	if subm == nil {
		return true
	}
	if subm.CgOomKilled || subm.RamKiBytes > int64(ramKiB) {
		return true
	}
	if subm.CpuMillis > int64(cpuMs) {
		return true
	}
	if subm.WallMillis > wallLimitMs {
		return true
	}
	if subm.ExitSignal != nil || subm.ExitCode != 0 || len(subm.Stderr) > 0 {
		return true
	}
	if chkr == nil || chkr.ExitCode != 0 {
		return true
	}
	return false
}
