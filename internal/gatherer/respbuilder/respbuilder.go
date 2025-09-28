package respbuilder

import (
	"time"

	"github.com/programme-lv/tester/api"
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
func (b *Builder) FinishCompile(data *api.RuntimeData) {
	if data == nil {
		b.compileRun = nil
		return
	}
	b.compileRun = data
}

// ReachTest implements ResultGatherer.
func (b *Builder) ReachTest(testId int64, input []byte, answer []byte) {}

// IgnoreTest implements ResultGatherer.
func (b *Builder) IgnoreTest(testId int64) {
	// Represent ignored test as a result with no runtime data
	b.testResults = append(b.testResults, api.TestResult{TestId: int32(testId)})
}

// FinishTest implements ResultGatherer.
func (b *Builder) FinishTest(testId int64, subm *api.RuntimeData, chkr *api.RuntimeData) {
	tr := api.TestResult{TestId: int32(testId)}
	tr.Subm = subm
	tr.Chkr = chkr
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
		ErrorMsg:    b.errorMessage,
		StartTime:   start,
		FinishTime:  finish,
		TotalTimeMs: total,
		SystemInfo:  b.systemInfo,
	}
}
