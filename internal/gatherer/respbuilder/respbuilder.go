package respbuilder

import (
	"time"

	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal"
)

// Builder gathers execution events and builds a complete api.ExecResponse.
type Builder struct {
	evalUuid   string
	systemInfo string

	started  time.Time
	finished *time.Time

	// compilation runtime data
	compileRun *api.RuntimeData

	// tests
	testResults []api.TestResult

	// job status
	status       api.ExecStatus
	errorMessage *string
}

func New(evalUuid string) *Builder {
	return &Builder{
		evalUuid: evalUuid,
		started:  time.Now(),
		status:   api.Success,
	}
}

// StartJob implements ResultGatherer.
func (b *Builder) StartJob(systemInfo string) {
	b.systemInfo = systemInfo
}

// StartCompile implements ResultGatherer.
func (b *Builder) StartCompile() {}

// FinishCompile implements ResultGatherer.
func (b *Builder) FinishCompile(data *internal.RunData) {
	if data == nil {
		b.compileRun = nil
		return
	}
	rd := &api.RuntimeData{
		CpuMillis:  int64(data.CpuMs),
		WallMillis: int64(data.WallMs),
		RamKiBytes: int64(data.MemKiB),
		ExitCode:   int64(data.ExitCode),
		Stdout:     string(data.Stdout),
		Stderr:     string(data.Stderr),
	}
	if data.ExitSignal != nil {
		sig := *data.ExitSignal
		rd.ExitSignal = &sig
	}
	b.compileRun = rd
}

// ReachTest implements ResultGatherer.
func (b *Builder) ReachTest(testId int64, input []byte, answer []byte) {}

// IgnoreTest implements ResultGatherer.
func (b *Builder) IgnoreTest(testId int64) {
	// Represent ignored test as a result with no runtime data
	b.testResults = append(b.testResults, api.TestResult{TestId: int32(testId)})
}

// FinishTest implements ResultGatherer.
func (b *Builder) FinishTest(testId int64, subm *internal.RunData, chkr *internal.RunData) {
	tr := api.TestResult{TestId: int32(testId)}
	// Map helper
	mapRun := func(src *internal.RunData) *api.RuntimeData {
		if src == nil {
			return nil
		}
		rd := &api.RuntimeData{}
		rd.CpuMillis = int64(src.CpuMs)
		rd.WallMillis = int64(src.WallMs)
		rd.RamKiBytes = int64(src.MemKiB)
		rd.ExitCode = int64(src.ExitCode)
		if src.ExitSignal != nil {
			sig := *src.ExitSignal
			rd.ExitSignal = &sig
		}
		rd.Stdout = string(src.Stdout)
		rd.Stderr = string(src.Stderr)
		if src.IsolateStatus != nil {
			msg := *src.IsolateStatus
			if src.IsolateMsg != nil {
				msg += ": " + *src.IsolateMsg
			}
			rd.IsolateStatus = &msg
			rd.IsolateMsg = src.IsolateMsg
		}
		return rd
	}
	tr.Subm = mapRun(subm)
	tr.Chkr = mapRun(chkr)
	b.testResults = append(b.testResults, tr)
}

// CompileError implements ResultGatherer.
func (b *Builder) CompileError(msg string) {
	b.status = api.CompileError
	b.errorMessage = &msg
}

// InternalError implements ResultGatherer.
func (b *Builder) InternalError(msg string) {
	b.status = api.InternalError
	b.errorMessage = &msg
}

// FinishNoError implements ResultGatherer.
func (b *Builder) FinishNoError() {
	now := time.Now()
	b.finished = &now
}

// Response builds the api.ExecResponse from gathered data.
func (b *Builder) Response() api.ExecResponse {
	start := b.started.Format(time.RFC3339)
	finish := start
	total := int64(0)
	if b.finished != nil {
		finish = b.finished.Format(time.RFC3339)
		total = b.finished.Sub(b.started).Milliseconds()
	}
	return api.ExecResponse{
		EvalUuid:    b.evalUuid,
		Status:      b.status,
		Compilation: b.compileRun,
		TestResults: b.testResults,
		ErrorMsg: func() *string {
			if b.errorMessage == nil {
				return nil
			}
			v := *b.errorMessage
			return &v
		}(),
		StartTime:   start,
		FinishTime:  finish,
		TotalTimeMs: total,
		SystemInfo: func() *string {
			if b.systemInfo == "" {
				return nil
			}
			v := b.systemInfo
			return &v
		}(),
	}
}
