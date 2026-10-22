package tester

import (
	"syscall"
	"testing"

	"github.com/programme-lv/tester/api"
)

func TestGroupSkipNoGroupsRunsAll(t *testing.T) {
	s := newGroupSkip(nil)
	s.markFailed(1)
	if s.shouldIgnore(2) {
		t.Fatal("empty groups must not ignore later tests")
	}
}

func TestGroupSkipDisjoint(t *testing.T) {
	s := newGroupSkip([][]int{{1, 2}, {3, 4}})
	if s.shouldIgnore(1) {
		t.Fatal("first test of a live group must run")
	}
	s.markFailed(1)
	if !s.shouldIgnore(2) {
		t.Fatal("rest of failed group must be ignored")
	}
	if s.shouldIgnore(3) {
		t.Fatal("other group must still run")
	}
	s.markFailed(3)
	if !s.shouldIgnore(4) {
		t.Fatal("rest of second failed group must be ignored")
	}
}

func TestGroupSkipOverlappingKeepsLiveUnit(t *testing.T) {
	s := newGroupSkip([][]int{{1, 2}, {2, 3}})
	s.markFailed(1)
	if s.shouldIgnore(2) {
		t.Fatal("test still needed by a live group must run")
	}
	s.markFailed(2)
	if !s.shouldIgnore(3) {
		t.Fatal("test whose every group is dead must be ignored")
	}
}

func TestGroupSkipUngroupedAlwaysRuns(t *testing.T) {
	s := newGroupSkip([][]int{{1, 2}})
	s.markFailed(1)
	if s.shouldIgnore(3) {
		t.Fatal("test in no group must run")
	}
}

func TestTestFailed(t *testing.T) {
	ok := &api.RuntimeData{}
	if testFailed(ok, ok, 1000, 1024) {
		t.Fatal("accepted run must not fail")
	}
	if !testFailed(nil, ok, 1000, 1024) {
		t.Fatal("nil submission must fail")
	}
	if !testFailed(ok, nil, 1000, 1024) {
		t.Fatal("nil checker must fail")
	}
	wa := &api.RuntimeData{ExitCode: 1}
	if !testFailed(ok, wa, 1000, 1024) {
		t.Fatal("checker nonzero exit must fail")
	}
	tle := &api.RuntimeData{CpuMillis: 1001}
	if !testFailed(tle, ok, 1000, 1024) {
		t.Fatal("cpu over limit must fail")
	}
	mle := &api.RuntimeData{RamKiBytes: 1025}
	if !testFailed(mle, ok, 1000, 1024) {
		t.Fatal("memory over limit must fail")
	}
	sig := int64(9)
	re := &api.RuntimeData{ExitSignal: &sig}
	if !testFailed(re, ok, 1000, 1024) {
		t.Fatal("signal must fail")
	}
}

func TestSubmissionStderrOnlyFailsCheckerTasks(t *testing.T) {
	withStderr := &api.RuntimeData{Stderr: "warning"}
	if !submissionFailed(withStderr, 1000, 1024, true) {
		t.Fatal("checker-based tasks keep failing on stderr")
	}
	if submissionFailed(withStderr, 1000, 1024, false) {
		t.Fatal("interactive tasks must ignore stderr")
	}
}

func TestSubmissionTimeoutStatusFails(t *testing.T) {
	status := "TO"
	to := &api.RuntimeData{IsolateStatus: &status}
	if !submissionFailed(to, 1000, 1024, false) {
		t.Fatal("isolate TO status must fail")
	}
}

func TestClassifyInteractor(t *testing.T) {
	if outcome, _ := classifyInteractor(&api.RuntimeData{}); outcome != interactorAccepted {
		t.Fatalf("exit 0 must be accepted, got %v", outcome)
	}
	for _, code := range []int64{1, 2, 8} {
		chkr := &api.RuntimeData{ExitCode: code}
		if outcome, _ := classifyInteractor(chkr); outcome != interactorRejected {
			t.Fatalf("exit %d must reject, got %v", code, outcome)
		}
	}
	for _, code := range []int64{3, 7, 42, 151} {
		chkr := &api.RuntimeData{ExitCode: code}
		if outcome, _ := classifyInteractor(chkr); outcome != interactorBroken {
			t.Fatalf("exit %d must be a judge failure, got %v", code, outcome)
		}
	}
	for code, percent := range map[int64]int{50: 0, 90: 40, 150: 100} {
		chkr := &api.RuntimeData{ExitCode: code}
		outcome, _ := classifyInteractor(chkr)
		if outcome != interactorPartial {
			t.Fatalf("exit %d must be partial, got %v", code, outcome)
		}
		if got := interactorScorePercent(chkr); got != percent {
			t.Fatalf("exit %d must score %d%%, got %d%%", code, percent, got)
		}
	}
	pipe := int64(syscall.SIGPIPE)
	if outcome, _ := classifyInteractor(&api.RuntimeData{ExitSignal: &pipe}); outcome != interactorRejected {
		t.Fatal("SIGPIPE must reject")
	}
	segv := int64(syscall.SIGSEGV)
	if outcome, _ := classifyInteractor(&api.RuntimeData{ExitSignal: &segv}); outcome != interactorBroken {
		t.Fatal("SIGSEGV must be a judge failure")
	}
	status := "TO"
	if outcome, _ := classifyInteractor(&api.RuntimeData{IsolateStatus: &status}); outcome != interactorTimedOut {
		t.Fatal("TO must be a timeout")
	}
	status = "XX"
	if outcome, _ := classifyInteractor(&api.RuntimeData{IsolateStatus: &status}); outcome != interactorBroken {
		t.Fatal("XX must be a judge failure")
	}
	if outcome, _ := classifyInteractor(&api.RuntimeData{CgOomKilled: true}); outcome != interactorBroken {
		t.Fatal("interactor OOM must be a judge failure")
	}
}
