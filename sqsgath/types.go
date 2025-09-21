package sqsgath

import (
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal"
)

type sqsResQueueGatherer struct {
	sqsClient *sqs.Client
	queueUrl  string
	evalUuid  string
}

func (s *sqsResQueueGatherer) FinishCompilation(data *internal.RuntimeData) {
	msg := api.NewFinishedCompilation(
		s.evalUuid,
		mapRunData(data, api.MaxRuntimeDataHeight*2, api.MaxRuntimeDataWidth*2),
	)
	s.send(msg)
}

func (s *sqsResQueueGatherer) FinishEvalWithCompileError(msg string) {
	s.send(api.NewFinishedEvaluation(s.evalUuid, &msg, true, false))
}

func (s *sqsResQueueGatherer) FinishEvalWithInternalError(msg string) {
	s.send(api.NewFinishedEvaluation(s.evalUuid, &msg, false, true))
}

func (s *sqsResQueueGatherer) FinishEvalWithoutError() {
	s.send(api.NewFinishedEvaluation(s.evalUuid, nil, false, false))
}

func mapRunData(data *internal.RuntimeData, ioHeight int, ioWidth int) *api.RuntimeData {
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
		CpuMillis:     data.CpuMs,
		WallMillis:    data.WallMs,
		MemoryKiBytes: data.MemKiB,
		CtxSwV:        data.CtxSwV,
		CtxSwF:        data.CtxSwF,
		ExitSignal:    data.ExitSignal,
		IsolateStatus: data.IsolateStatus,
		IsolateMsg:    data.IsolateMsg,
	}
}

func (s *sqsResQueueGatherer) FinishTest(testId int64, submission *internal.RuntimeData, checker *internal.RuntimeData) {
	msg := api.NewFinishedTest(
		s.evalUuid,
		testId,
		mapRunData(submission, api.MaxRuntimeDataHeight, api.MaxRuntimeDataWidth),
		mapRunData(checker, api.MaxRuntimeDataHeight, api.MaxRuntimeDataWidth),
	)
	s.send(msg)
}

// FinishTesting implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) FinishTesting() {
	s.send(api.NewFinishedTesting(s.evalUuid))
}

// IgnoreTest implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) IgnoreTest(testId int64) {
	s.send(api.NewIgnoredTest(s.evalUuid, testId))
}

// StartCompilation implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) StartCompilation() {
	s.send(api.NewStartedCompilation(s.evalUuid))
}

// StartEvaluation implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) StartEvaluation(systemInfo string) {
	s.send(api.NewStartedEvaluation(s.evalUuid, systemInfo))
}

// ReachTest implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) ReachTest(testId int64, input []byte, answer []byte) {
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
	s.send(api.NewReachedTest(s.evalUuid, testId, inputStrPtr, answerStrPtr))
}

// StartTesting implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) StartTesting() {
	s.send(api.NewStartedTesting(s.evalUuid))
}
