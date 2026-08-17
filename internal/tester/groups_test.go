package tester

import (
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
