package termgath

import (
	"fmt"
	"time"

	"github.com/programme-lv/tester/internal"
)

type TerminalGatherer struct {
	StartedAt time.Time
}

func New() *TerminalGatherer { return &TerminalGatherer{StartedAt: time.Now()} }

func (t *TerminalGatherer) StartEvaluation(systemInfo string) {
	fmt.Println("== Evaluation started ==")
	if systemInfo != "" {
		fmt.Println("System info:")
		fmt.Println(systemInfo)
	}
}

func (t *TerminalGatherer) StartCompilation() {
	fmt.Println("-- Compilation started --")
}

func (t *TerminalGatherer) FinishCompilation(data *internal.RuntimeData) {
	fmt.Println("-- Compilation finished --")
	if data != nil {
		fmt.Printf("exit=%d cpu=%dms wall=%dms mem=%dKiB\n", data.ExitCode, data.CpuMs, data.WallMs, data.MemKiB)
		if len(data.Stderr) > 0 {
			fmt.Printf("stderr:\n%s\n", string(data.Stderr))
		}
	}
}

func (t *TerminalGatherer) ReachTest(testId int64, input []byte, answer []byte) {
	fmt.Printf("-> Test %d reached\n", testId)
}

func (t *TerminalGatherer) IgnoreTest(testId int64) {
	fmt.Printf("-> Test %d ignored\n", testId)
}

func (t *TerminalGatherer) FinishTest(testId int64, submission *internal.RuntimeData, checker *internal.RuntimeData) {
	fmt.Printf("<- Test %d finished\n", testId)
	if submission != nil {
		fmt.Printf("  subm: exit=%d cpu=%dms wall=%dms mem=%dKiB\n", submission.ExitCode, submission.CpuMs, submission.WallMs, submission.MemKiB)
	}
	if checker != nil {
		fmt.Printf("  chkr: exit=%d cpu=%dms wall=%dms mem=%dKiB\n", checker.ExitCode, checker.CpuMs, checker.WallMs, checker.MemKiB)
	}
}

func (t *TerminalGatherer) FinishEvalWithCompileError(msg string) {
	fmt.Printf("== Compilation error: %s ==\n", msg)
}

func (t *TerminalGatherer) FinishEvalWithInternalError(msg string) {
	fmt.Printf("== Internal error: %s ==\n", msg)
}

func (t *TerminalGatherer) FinishEvalWithoutError() {
	dur := time.Since(t.StartedAt).Round(time.Millisecond)
	fmt.Printf("== Evaluation finished in %s ==\n", dur)
}
