package sqsgath

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/programme-lv/tester/internal"
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
	RuntimeData *internal.RuntimeData `json:"runtime_data"`
}

const (
	MaxRuntimeDataHeight = 40
	MaxRuntimeDataWidth  = 80
)

func (s *sqsResQueueGatherer) FinishCompilation(data *internal.RuntimeData) {
	header := Header{
		EvalUuid: s.evalUuid,
		MsgType:  MsgTypeFinishedCompilation,
	}
	msg := FinishedCompilation{
		Header:      header,
		RuntimeData: trimRuntimeData(data, MaxRuntimeDataHeight, MaxRuntimeDataWidth),
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
	TestId     int64                 `json:"test_id"`
	Submission *internal.RuntimeData `json:"submission"`
	Checker    *internal.RuntimeData `json:"checker"`
}

func (s *sqsResQueueGatherer) FinishTest(testId int64, submission *internal.RuntimeData, checker *internal.RuntimeData) {
	msg := FinishedTest{
		Header: Header{
			EvalUuid: s.evalUuid,
			MsgType:  MsgTypeFinishedTest,
		},
		TestId:     testId,
		Submission: trimRuntimeData(submission, MaxRuntimeDataHeight, MaxRuntimeDataWidth),
		Checker:    trimRuntimeData(checker, MaxRuntimeDataHeight, MaxRuntimeDataWidth),
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
func (s *sqsResQueueGatherer) ReachTest(testId int64, input *string, answer *string) {
	header := Header{
		EvalUuid: s.evalUuid,
		MsgType:  MsgTypeReachedTest,
	}
	msg := ReachedTest{
		Header: header,
		TestId: testId,
		Input:  trimStringToRectangle(input, MaxRuntimeDataHeight, MaxRuntimeDataWidth),
		Answer: trimStringToRectangle(answer, MaxRuntimeDataHeight, MaxRuntimeDataWidth),
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
