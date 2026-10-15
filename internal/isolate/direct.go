package isolate

import (
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// Direct mode runs commands without the isolate sandbox for verifying the
// tester on machines that cannot run isolate. It is not a security boundary
// and must never run untrusted submissions.
var directMode atomic.Bool

func SetDirectMode(enabled bool) {
	directMode.Store(enabled)
}

func DirectMode() bool {
	return directMode.Load()
}

func newDirectBox() (*Box, error) {
	root, err := os.MkdirTemp("", "tester-box-")
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(filepath.Join(root, "box"), 0o777)
	if err != nil {
		os.RemoveAll(root)
		return nil, err
	}
	return &Box{id: -1, path: root, direct: true}, nil
}

func directCommand(box *Box, cmd *Cmd, command string) *Cmd {
	goCmd := exec.Command("/usr/bin/bash", "-c", directScript(command, cmd.Constraints))
	goCmd.Dir = filepath.Join(box.path, "box")
	// In its own process group so the whole tree can be killed on timeout.
	goCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.cmd = goCmd
	cmd.direct = true
	return cmd
}

// directScript wraps the command with best-effort limits. Address-space
// limits below 256 MiB are raised, because RLIMIT_AS breaks dynamic loaders
// where isolate's cgroup limit would not.
func directScript(command string, c Constraints) string {
	limits := []string{"ulimit -c 0"}
	memKb := c.MemoryLimitInKB
	if memKb > 0 && memKb < 256*1024 {
		memKb = 256 * 1024
	}
	if memKb > 0 {
		limits = append(limits, fmt.Sprintf("ulimit -v %d", memKb))
	}
	cpuSecs := int(math.Ceil(c.CpuTimeLimInSec + c.ExtraCpuTimeLimInSec))
	if cpuSecs < 1 {
		cpuSecs = 1
	}
	limits = append(limits, fmt.Sprintf("ulimit -t %d", cpuSecs))
	// RLIMIT_NPROC counts the whole UID, not the command tree, so isolate's
	// per-box process cap is not reproduced here.
	if c.MaxOpenFiles > 0 {
		limits = append(limits, fmt.Sprintf("ulimit -n %d", c.MaxOpenFiles))
	}
	return fmt.Sprintf("%s 2>/dev/null; exec /usr/bin/bash -c %q", strings.Join(limits, "; "), command)
}

func (process *Cmd) waitDirect() (*Metrics, error) {
	var fired chan struct{}
	var timer *time.Timer
	wall := time.Duration(process.Constraints.WallTimeLimInSec * float64(time.Second))
	if wall > 0 {
		fired = make(chan struct{})
		timer = time.AfterFunc(wall, func() {
			process.timedOut.Store(true)
			killProcessGroup(process.cmd)
			close(fired)
		})
	}
	err := process.cmd.Wait()
	if timer != nil && !timer.Stop() {
		<-fired
	}
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, err
		}
	}
	return directMetrics(process), nil
}

// directMetrics synthesizes isolate-style metrics from the process exit
// status and rusage. Memory is deliberately not reported: max RSS of the
// shell tree is not comparable to isolate's cgroup accounting, so memory
// checks are skipped and MLE verdicts cannot be verified without isolate.
func directMetrics(process *Cmd) *Metrics {
	state := process.cmd.ProcessState
	m := &Metrics{ExitCode: int64(state.ExitCode())}
	if m.ExitCode < 0 {
		m.ExitCode = 0
	}
	if ws, ok := state.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		sig := int64(ws.Signal())
		m.ExitSig = &sig
	}
	if usage, ok := state.SysUsage().(*syscall.Rusage); ok {
		m.CpuMillis = timevalMillis(usage.Utime) + timevalMillis(usage.Stime)
		m.MaxRssKb = int64(usage.Maxrss)
	}
	m.WallMillis = time.Since(process.startedAt).Milliseconds()
	switch {
	case process.timedOut.Load():
		setStatus(m, "TO", "wall time limit exceeded (no-isolate mode)")
	case m.ExitSig != nil && *m.ExitSig == int64(syscall.SIGXCPU):
		setStatus(m, "TO", "cpu time limit exceeded (no-isolate mode)")
	case m.ExitCode != 0:
		setStatus(m, "RE", "")
	}
	m.FullReport = fmt.Sprintf("direct mode: exit=%d cpu=%dms wall=%dms maxrss=%dKB",
		m.ExitCode, m.CpuMillis, m.WallMillis, m.MaxRssKb)
	return m
}

func setStatus(m *Metrics, status, message string) {
	m.Status = &status
	if message != "" {
		m.Message = &message
	}
	m.Killed = 1
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	_ = cmd.Process.Kill()
}

func timevalMillis(tv syscall.Timeval) int64 {
	return tv.Sec*1000 + tv.Usec/1000
}
