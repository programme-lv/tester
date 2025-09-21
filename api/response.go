package api

import "time"

// MsgType represents a message type as a string alias
type MsgType string

// Message type constants
const (
	ReceiveJob     MsgType = "job_receive"
	StartCompile   MsgType = "compile_start"
	FinishCompile  MsgType = "compile_finish"
	ReachTestCase  MsgType = "test_reach"
	IgnoreTestCase MsgType = "test_ignore"
	FinishTestCase MsgType = "test_finish"
	FinishJob      MsgType = "job_finish"
)

// Runtime data size constraints
const (
	MaxRuntimeDataHeight = 40
	MaxRuntimeDataWidth  = 80
)

// Header is the common header for all response messages
type Header struct {
	EvalUuid string  `json:"eval_uuid"`
	MsgType  MsgType `json:"msg_type"`
}

// RuntimeData contains execution information for a process
type RuntimeData struct {
	Stdin    string `json:"in"`
	Stdout   string `json:"out"`
	Stderr   string `json:"err"`
	ExitCode int64  `json:"exit"`

	CpuMillis     int64 `json:"cpu_ms"`
	WallMillis    int64 `json:"wall_ms"`
	MemoryKiBytes int64 `json:"mem_kib"`

	CtxSwV int64 `json:"ctx_sw_v"`
	CtxSwF int64 `json:"ctx_sw_f"`

	ExitSignal    *int64  `json:"signal"`
	IsolateStatus *string `json:"isolate_status"`
	IsolateMsg    *string `json:"isolate_msg"`
}

// StartedEvaluation message sent when evaluation begins
type StartedEvaluation struct {
	Header
	SystemInfo  string `json:"system_info"`
	StartedTime string `json:"started_time"`
}

// StartedCompilation message sent when compilation begins
type StartedCompilation struct {
	Header
}

// FinishedCompilation message sent when compilation completes
type FinishedCompilation struct {
	Header
	RuntimeData *RuntimeData `json:"runtime_data"`
}

// StartedTesting message sent when testing begins
type StartedTesting struct {
	Header
}

// ReachedTest message sent when a test is reached
type ReachedTest struct {
	Header
	TestId int64   `json:"test_id"`
	Input  *string `json:"input"`
	Answer *string `json:"answer"`
}

// IgnoredTest message sent when a test is ignored
type IgnoredTest struct {
	Header
	TestId int64 `json:"test_id"`
}

// FinishedTest message sent when a test completes
type FinishedTest struct {
	Header
	TestId     int64        `json:"test_id"`
	Submission *RuntimeData `json:"submission"`
	Checker    *RuntimeData `json:"checker"`
}

// FinishedEvaluation message sent when evaluation completes
type FinishedEvaluation struct {
	Header
	ErrorMessage  *string `json:"error_message"`
	CompileError  bool    `json:"compile_error"`
	InternalError bool    `json:"internal_error"`
}

// Helper function to create a header
func NewHeader(evalUuid string, msgType MsgType) Header {
	return Header{
		EvalUuid: evalUuid,
		MsgType:  msgType,
	}
}

// Helper functions to create specific message types
func NewStartedEvaluation(evalUuid, systemInfo string) StartedEvaluation {
	return StartedEvaluation{
		Header:      NewHeader(evalUuid, ReceiveJob),
		SystemInfo:  systemInfo,
		StartedTime: time.Now().Format(time.RFC3339),
	}
}

func NewStartedCompilation(evalUuid string) StartedCompilation {
	return StartedCompilation{
		Header: NewHeader(evalUuid, StartCompile),
	}
}

func NewFinishedCompilation(evalUuid string, runtimeData *RuntimeData) FinishedCompilation {
	return FinishedCompilation{
		Header:      NewHeader(evalUuid, FinishCompile),
		RuntimeData: runtimeData,
	}
}

func NewReachedTest(evalUuid string, testId int64, input, answer *string) ReachedTest {
	return ReachedTest{
		Header: NewHeader(evalUuid, ReachTestCase),
		TestId: testId,
		Input:  input,
		Answer: answer,
	}
}

func NewIgnoredTest(evalUuid string, testId int64) IgnoredTest {
	return IgnoredTest{
		Header: NewHeader(evalUuid, IgnoreTestCase),
		TestId: testId,
	}
}

func NewFinishedTest(evalUuid string, testId int64, submission, checker *RuntimeData) FinishedTest {
	return FinishedTest{
		Header:     NewHeader(evalUuid, FinishTestCase),
		TestId:     testId,
		Submission: submission,
		Checker:    checker,
	}
}

func NewFinishedEvaluation(evalUuid string, errorMessage *string, compileError, internalError bool) FinishedEvaluation {
	return FinishedEvaluation{
		Header:        NewHeader(evalUuid, FinishJob),
		ErrorMessage:  errorMessage,
		CompileError:  compileError,
		InternalError: internalError,
	}
}
