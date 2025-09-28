package api

import "time"

// MsgType is a message type for streaming responses
type MsgType string

// Streaming message type constants
const (
	StartJobMsg      MsgType = "job_start"
	StartCompileMsg  MsgType = "compile_start"
	FinishCompileMsg MsgType = "compile_finish"
	ReachTestMsg     MsgType = "test_reach"
	IgnoreTestMsg    MsgType = "test_ignore"
	FinishTestMsg    MsgType = "test_finish"
	FinishJobMsg     MsgType = "job_finish"
)

// Runtime data size constraints for streaming
const (
	MaxRuntimeDataHeight = 40
	MaxRuntimeDataWidth  = 80
)

// Header is the common header for all streaming response messages
type Header struct {
	EvalUuid string  `json:"eval_uuid"`
	MsgType  MsgType `json:"msg_type"`
}

// RuntimeData contains execution information for a process (streaming version)
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

// StartJob message sent when evaluation begins
type StartJob struct {
	Header
	SystemInfo  string `json:"system_info"`
	StartedTime string `json:"started_time"`
}

// StartCompile message sent when compilation begins
type StartCompile struct {
	Header
}

// FinishCompile message sent when compilation completes
type FinishCompile struct {
	Header
	RuntimeData *RuntimeData `json:"runtime_data"`
}

// ReachTest message sent when a test is reached
type ReachTest struct {
	Header
	TestId int64   `json:"test_id"`
	Input  *string `json:"input"`
	Answer *string `json:"answer"`
}

// IgnoreTest message sent when a test is ignored
type IgnoreTest struct {
	Header
	TestId int64 `json:"test_id"`
}

// FinishTest message sent when a test completes
type FinishTest struct {
	Header
	TestId     int64        `json:"test_id"`
	Submission *RuntimeData `json:"submission"`
	Checker    *RuntimeData `json:"checker"`
}

// FinishJob message sent when evaluation completes
type FinishJob struct {
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

// Helper functions to create specific streaming message types
func NewStartJob(evalUuid, systemInfo string) StartJob {
	return StartJob{
		Header:      NewHeader(evalUuid, StartJobMsg),
		SystemInfo:  systemInfo,
		StartedTime: time.Now().Format(time.RFC3339),
	}
}

func NewStartCompile(evalUuid string) StartCompile {
	return StartCompile{
		Header: NewHeader(evalUuid, StartCompileMsg),
	}
}

func NewFinishCompile(evalUuid string, runtimeData *RuntimeData) FinishCompile {
	return FinishCompile{
		Header:      NewHeader(evalUuid, FinishCompileMsg),
		RuntimeData: runtimeData,
	}
}

func NewReachTest(evalUuid string, testId int64, input, answer *string) ReachTest {
	return ReachTest{
		Header: NewHeader(evalUuid, ReachTestMsg),
		TestId: testId,
		Input:  input,
		Answer: answer,
	}
}

func NewIgnoreTest(evalUuid string, testId int64) IgnoreTest {
	return IgnoreTest{
		Header: NewHeader(evalUuid, IgnoreTestMsg),
		TestId: testId,
	}
}

func NewFinishTest(evalUuid string, testId int64, submission, checker *RuntimeData) FinishTest {
	return FinishTest{
		Header:     NewHeader(evalUuid, FinishTestMsg),
		TestId:     testId,
		Submission: submission,
		Checker:    checker,
	}
}

func NewFinishJob(evalUuid string, errorMessage *string, compileError, internalError bool) FinishJob {
	return FinishJob{
		Header:        NewHeader(evalUuid, FinishJobMsg),
		ErrorMessage:  errorMessage,
		CompileError:  compileError,
		InternalError: internalError,
	}
}
