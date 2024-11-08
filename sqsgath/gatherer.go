package sqsgath

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	pkg "github.com/programme-lv/tester/pkg"
)

type sqsResQueueGatherer struct {
	sqsClient *sqs.Client
	queueUrl  string
	evalUuid  string
}

const (
	MsgTypeStartedEvaluation   = "started_evaluation"
	MsgTypeStartedCompilation  = "started_compilation"
	MsgTypeFinishedCompilation = "finished_compilation"
	MsgTypeStartedTesting      = "started_testing"
	MsgTypeReachedTest         = "reached_test"
	MsgTypeIgnoredTest         = "ignored_test"
	MsgTypeFinishedTest        = "finished_test"
	MsgTypeFinishedTesting     = "finished_testing"
	MsgTypeFinishedEvaluation  = "finished_evaluation"
)

type Header struct {
	EvalUuid string `json:"eval_uuid"`
	MsgType  string `json:"msg_type"`
}

type FinishedCompilation struct {
	Header
	RuntimeData *RuntimeData `json:"runtime_data"`
}

const (
	MaxRuntimeDataHeight = 40
	MaxRuntimeDataWidth  = 80
)

func (s *sqsResQueueGatherer) FinishCompilation(data *pkg.RuntimeData) {
	header := Header{
		EvalUuid: s.evalUuid,
		MsgType:  MsgTypeFinishedCompilation,
	}
	msg := FinishedCompilation{
		Header:      header,
		RuntimeData: mapRunData(data),
	}
	s.send(msg)
}

type FinishedEvaluation struct {
	Header
	ErrorMessage  *string `json:"error_message"`
	CompileError  bool    `json:"compile_error"`
	InternalError bool    `json:"internal_error"`
}

func (s *sqsResQueueGatherer) FinishEvalWithCompileError(msg string) {
	header := Header{
		EvalUuid: s.evalUuid,
		MsgType:  MsgTypeFinishedEvaluation,
	}
	s.send(FinishedEvaluation{
		Header:        header,
		ErrorMessage:  &msg,
		CompileError:  true,
		InternalError: false,
	})
}

func (s *sqsResQueueGatherer) FinishEvalWithInternalError(msg string) {
	header := Header{
		EvalUuid: s.evalUuid,
		MsgType:  MsgTypeFinishedEvaluation,
	}
	s.send(FinishedEvaluation{
		Header:        header,
		ErrorMessage:  &msg,
		CompileError:  false,
		InternalError: true,
	})
}

func (s *sqsResQueueGatherer) FinishEvalWithoutError() {
	header := Header{
		EvalUuid: s.evalUuid,
		MsgType:  MsgTypeFinishedEvaluation,
	}
	s.send(FinishedEvaluation{
		Header:        header,
		ErrorMessage:  nil,
		CompileError:  false,
		InternalError: false,
	})
}

type FinishedTest struct {
	Header
	TestId     int64        `json:"test_id"`
	Submission *RuntimeData `json:"submission"`
	Checker    *RuntimeData `json:"checker"`
}

type RuntimeData struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int64  `json:"exit_code"`

	CpuMillis     int64 `json:"cpu_time_millis"`
	WallMillis    int64 `json:"wall_time_millis"`
	MemoryKiBytes int64 `json:"memory_kibibytes"`

	CtxSwVoluntary int64 `json:"context_switches_voluntary"`
	CtxSwForced    int64 `json:"context_switches_forced"`

	ExitSignal    *int64 `json:"exit_signal"`
	IsolateStatus string `json:"isolate_status"`
}

func mapRunData(data *pkg.RuntimeData) *RuntimeData {
	if data == nil {
		return nil
	}
	var stdout string = ""
	if len(data.Stdout) > 0 {
		stdout = string(data.Stdout)
	}
	var stderr string = ""
	if len(data.Stderr) > 0 {
		stderr = string(data.Stderr)
	}
	return &RuntimeData{
		Stdout:         trimStrToRect(stdout, MaxRuntimeDataHeight, MaxRuntimeDataWidth),
		Stderr:         trimStrToRect(stderr, MaxRuntimeDataHeight, MaxRuntimeDataWidth),
		ExitCode:       data.ExitCode,
		CpuMillis:      data.CpuMillis,
		WallMillis:     data.WallMillis,
		MemoryKiBytes:  data.MemoryKiBytes,
		CtxSwVoluntary: data.CtxSwVoluntary,
		CtxSwForced:    data.CtxSwForced,
		ExitSignal:     data.ExitSignal,
		IsolateStatus:  data.IsolateStatus,
	}
}

func (s *sqsResQueueGatherer) FinishTest(testId int64, submission *pkg.RuntimeData, checker *pkg.RuntimeData) {
	msg := FinishedTest{
		Header: Header{
			EvalUuid: s.evalUuid,
			MsgType:  MsgTypeFinishedTest,
		},
		TestId:     testId,
		Submission: mapRunData(submission),
		Checker:    mapRunData(checker),
	}
	s.send(msg)
}

type FinishedTesting struct {
	Header
}

// FinishTesting implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) FinishTesting() {
	msg := FinishedTesting{
		Header: Header{
			EvalUuid: s.evalUuid,
			MsgType:  MsgTypeFinishedTesting,
		},
	}
	s.send(msg)
}

type IgnoredTest struct {
	Header
	TestId int64 `json:"test_id"`
}

// IgnoreTest implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) IgnoreTest(testId int64) {
	msg := IgnoredTest{
		Header: Header{
			EvalUuid: s.evalUuid,
			MsgType:  MsgTypeIgnoredTest,
		},
		TestId: testId,
	}
	s.send(msg)
}

type StartedCompilation struct {
	Header
}

// StartCompilation implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) StartCompilation() {
	msg := StartedCompilation{
		Header: Header{
			EvalUuid: s.evalUuid,
			MsgType:  MsgTypeStartedCompilation,
		},
	}
	s.send(msg)
}

type StartedEvaluation struct {
	Header
	SystemInfo  string `json:"system_info"`
	StartedTime string `json:"started_time"`
}

// StartEvaluation implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) StartEvaluation(systemInfo string) {
	header := Header{
		EvalUuid: s.evalUuid,
		MsgType:  MsgTypeStartedEvaluation,
	}
	msg := StartedEvaluation{
		Header:      header,
		SystemInfo:  systemInfo,
		StartedTime: time.Now().Format(time.RFC3339),
	}
	s.send(msg)
}

type ReachedTest struct {
	Header
	TestId int64   `json:"test_id"`
	Input  *string `json:"input"`
	Answer *string `json:"answer"`
}

// ReachTest implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) ReachTest(testId int64, input []byte, answer []byte) {
	header := Header{
		EvalUuid: s.evalUuid,
		MsgType:  MsgTypeReachedTest,
	}
	var inputStrPtr *string = nil
	trimmedInput := trimStrToRect(string(input), MaxRuntimeDataHeight, MaxRuntimeDataWidth)
	if trimmedInput != "" {
		inputStr := trimmedInput
		inputStrPtr = &inputStr
	}
	var answerStrPtr *string = nil
	trimmedAnswer := trimStrToRect(string(answer), MaxRuntimeDataHeight, MaxRuntimeDataWidth)
	if trimmedAnswer != "" {
		answerStr := trimmedAnswer
		answerStrPtr = &answerStr
	}
	msg := ReachedTest{
		Header: header,
		TestId: testId,
		Input:  inputStrPtr,
		Answer: answerStrPtr,
	}
	s.send(msg)
}

type StartedTesting struct {
	Header
}

// StartTesting implements tester.EvalResGatherer.
func (s *sqsResQueueGatherer) StartTesting() {
	header := Header{
		EvalUuid: s.evalUuid,
		MsgType:  MsgTypeStartedTesting,
	}
	msg := StartedTesting{
		Header: header,
	}
	s.send(msg)
}
