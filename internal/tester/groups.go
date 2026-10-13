package tester

import (
	"fmt"
	"syscall"

	"github.com/programme-lv/tester/api"
)

// wallLimitMs is the effective wall-clock limit for the solution and the
// interactor. Isolate is configured to kill at the same threshold, so a
// timeout can surface either as a wall time over the limit or as an isolate
// TO status.
const wallLimitMs = 20000

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

// testFailed reports whether a checker-based test failed. The solution's
// stderr keeps its historical rule of failing the test.
func testFailed(subm, chkr *api.RuntimeData, cpuMs, ramKiB int32) bool {
	if submissionFailed(subm, cpuMs, ramKiB, true) {
		return true
	}
	if chkr == nil || chkr.ExitCode != 0 {
		return true
	}
	return false
}

// submissionFailed applies the solution's resource limits and termination
// rules. stderrFails is set for checker-based tasks and cleared for
// interactive tasks, where stderr is diagnostic only.
func submissionFailed(subm *api.RuntimeData, cpuMs, ramKiB int32, stderrFails bool) bool {
	if subm == nil {
		return true
	}
	if subm.CgOomKilled || subm.RamKiBytes > int64(ramKiB) {
		return true
	}
	if subm.CpuMillis > int64(cpuMs) {
		return true
	}
	if subm.WallMillis > wallLimitMs || isolateStatusIs(subm, "TO") {
		return true
	}
	if subm.ExitSignal != nil || subm.ExitCode != 0 {
		return true
	}
	if stderrFails && len(subm.Stderr) > 0 {
		return true
	}
	return false
}

// interactorOutcome classifies the interactor's termination.
type interactorOutcome int

const (
	interactorAccepted interactorOutcome = iota
	interactorRejected
	interactorTimedOut
	interactorBroken
)

// classifyInteractor maps the interactor's termination to the TaskZip
// verdict rules: exit 1 (WA), 2 (PE), and 8 (unexpected EOF), as well as
// death by SIGPIPE or SIGXFSZ, reject the solution; exit 0 accepts it;
// anything else is a judge failure. A timeout is classified separately,
// because it rejects the solution only while the solution is still running
// and is a judge failure once the solution has finished successfully.
func classifyInteractor(chkr *api.RuntimeData) (interactorOutcome, string) {
	if chkr == nil {
		return interactorBroken, "missing interactor runtime data"
	}
	if chkr.CgOomKilled {
		return interactorBroken, "interactor ran out of memory"
	}
	if isolateStatusIs(chkr, "TO") {
		return interactorTimedOut, "interactor timed out"
	}
	if chkr.ExitSignal != nil {
		sig := *chkr.ExitSignal
		if sig == int64(syscall.SIGPIPE) || sig == int64(syscall.SIGXFSZ) {
			return interactorRejected, fmt.Sprintf("interactor killed by signal %d", sig)
		}
		return interactorBroken, fmt.Sprintf("interactor killed by signal %d", sig)
	}
	if chkr.IsolateStatus != nil && *chkr.IsolateStatus != "RE" {
		return interactorBroken, fmt.Sprintf("interactor isolate status %s", *chkr.IsolateStatus)
	}
	switch chkr.ExitCode {
	case 0:
		return interactorAccepted, ""
	case 1, 2, 8:
		return interactorRejected, fmt.Sprintf("interactor exit code %d", chkr.ExitCode)
	default:
		return interactorBroken, fmt.Sprintf("interactor exit code %d", chkr.ExitCode)
	}
}

func isolateStatusIs(data *api.RuntimeData, status string) bool {
	return data.IsolateStatus != nil && *data.IsolateStatus == status
}
