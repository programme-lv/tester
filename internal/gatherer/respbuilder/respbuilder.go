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

	// compilation
	compileResult api.CompileResult

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
	// default to success unless exit != 0
	b.compileResult.Success = true
	if data != nil {
		if data.ExitCode != 0 {
			b.compileResult.Success = false
			msg := "compilation failed"
			if len(data.Stderr) > 0 {
				msg = string(data.Stderr)
			}
			b.compileResult.Error = &msg
		}
		cpu := int64(data.CpuMs)
		wall := int64(data.WallMs)
		mem := int64(data.MemKiB)
		b.compileResult.CpuMillis = &cpu
		b.compileResult.WallMillis = &wall
		b.compileResult.MemoryKiBytes = &mem
	}
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
	// Prefer checker data when present; otherwise use submission
	src := chkr
	if src == nil {
		src = subm
	}
	if src != nil {
		cpu := int64(src.CpuMs)
		wall := int64(src.WallMs)
		mem := int64(src.MemKiB)
		tr.CpuMillis = &cpu
		tr.WallMillis = &wall
		tr.MemoryKiBytes = &mem
		if src.ExitSignal != nil {
			sig := *src.ExitSignal
			tr.ExitSignal = &sig
		}
		code := src.ExitCode
		tr.ExitCode = &code
		if len(src.Stdout) > 0 {
			out := string(src.Stdout)
			tr.Stdout = &out
		}
		if len(src.Stderr) > 0 {
			err := string(src.Stderr)
			tr.Stderr = &err
		}
		if src.IsolateStatus != nil || src.IsolateMsg != nil {
			// If isolate reported an error, surface it on ErrorMessage
			if src.IsolateStatus != nil {
				msg := *src.IsolateStatus
				if src.IsolateMsg != nil {
					msg += ": " + *src.IsolateMsg
				}
				tr.ErrorMessage = &msg
			}
		}
	}
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
		Compilation: b.compileResult,
		TestResults: b.testResults,
		ErrorMessage: func() *string {
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
