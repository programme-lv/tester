package natsgath

import (
	"github.com/nats-io/nats.go"
	"github.com/programme-lv/tester/api"
)

type natsGatherer struct {
	nc       *nats.Conn
	inbox    string
	evalUuid string
}

func (s *natsGatherer) FinishCompile(data *api.RuntimeData) {
	msg := api.NewFinishCompile(
		s.evalUuid,
		trimRuntimeDataStrings(data, api.MaxRuntimeDataHeight, api.MaxRuntimeDataWidth),
	)
	s.send(msg)
}

func (s *natsGatherer) CompileError(msg string) {
	s.send(api.NewFinishJob(s.evalUuid, &msg, true, false))
}

func (s *natsGatherer) InternalError(msg string) {
	s.send(api.NewFinishJob(s.evalUuid, &msg, false, true))
}

func (s *natsGatherer) FinishNoError() {
	s.send(api.NewFinishJob(s.evalUuid, nil, false, false))
}

func trimRuntimeDataStrings(data *api.RuntimeData, ioHeight int, ioWidth int) *api.RuntimeData {
	if data == nil {
		return nil
	}
	var stdin string = ""
	if len(data.Stdin) > 0 {
		stdin = string(data.Stdin)
	}
	var stdout string = ""
	if len(data.Stdout) > 0 {
		stdout = string(data.Stdout)
	}
	var stderr string = ""
	if len(data.Stderr) > 0 {
		stderr = string(data.Stderr)
	}
	return &api.RuntimeData{
		Stdin:         trimStrToRect(stdin, ioHeight, ioWidth),
		Stdout:        trimStrToRect(stdout, ioHeight, ioWidth),
		Stderr:        trimStrToRect(stderr, ioHeight, ioWidth),
		ExitCode:      data.ExitCode,
		CpuMillis:     data.CpuMillis,
		WallMillis:    data.WallMillis,
		RamKiBytes:    data.RamKiBytes,
		CtxSwV:        data.CtxSwV,
		CtxSwF:        data.CtxSwF,
		ExitSignal:    data.ExitSignal,
		IsolateStatus: data.IsolateStatus,
		IsolateMsg:    data.IsolateMsg,
		CgOomKilled:   data.CgOomKilled,
	}
}

func (s *natsGatherer) FinishTest(testId int64, submission *api.RuntimeData, checker *api.RuntimeData) {
	msg := api.NewFinishTest(
		s.evalUuid,
		testId,
		trimRuntimeDataStrings(submission, api.MaxRuntimeDataHeight, api.MaxRuntimeDataWidth),
		trimRuntimeDataStrings(checker, api.MaxRuntimeDataHeight, api.MaxRuntimeDataWidth),
	)
	s.send(msg)
}

// IgnoreTest implements tester.EvalResGatherer.
func (s *natsGatherer) IgnoreTest(testId int64) {
	s.send(api.NewIgnoreTest(s.evalUuid, testId))
}

// StartCompilation implements tester.EvalResGatherer.
func (s *natsGatherer) StartCompile() {
	s.send(api.NewStartCompile(s.evalUuid))
}

// StartEvaluation implements tester.EvalResGatherer.
func (s *natsGatherer) StartJob(systemInfo string) {
	s.send(api.NewStartJob(s.evalUuid, systemInfo))
}

// ReachTest implements tester.EvalResGatherer.
func (s *natsGatherer) ReachTest(testId int64, input []byte, answer []byte) {
	var inputStrPtr *string = nil
	trimmedInput := trimStrToRect(string(input), api.MaxRuntimeDataHeight, api.MaxRuntimeDataWidth)
	if trimmedInput != "" {
		inputStr := trimmedInput
		inputStrPtr = &inputStr
	}
	var answerStrPtr *string = nil
	trimmedAnswer := trimStrToRect(string(answer), api.MaxRuntimeDataHeight, api.MaxRuntimeDataWidth)
	if trimmedAnswer != "" {
		answerStr := trimmedAnswer
		answerStrPtr = &answerStr
	}
	s.send(api.NewReachTest(s.evalUuid, testId, inputStrPtr, answerStrPtr))
}
