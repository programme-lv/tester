package messaging

type FeedbackType string

const (
	SubmissionReceived  FeedbackType = "submission_received"
	CompilationStarted  FeedbackType = "compilation_started"
	CompilationFinished FeedbackType = "compilation_finished"
	TestingStarted      FeedbackType = "testing_started"
	TestStarted         FeedbackType = "test_started"
	TestFinished        FeedbackType = "test_finished"
	IncrementScore      FeedbackType = "increment_score"
	FinishEvaluation    FeedbackType = "finish_evaluation"
)

type FeedbackData interface {
	isFeedbackData()
}

type SubmissionReceivedData struct {
	TestEnvInfo string `json:"test_env_info"`
	MaxScore    int    `json:"max_score"`
}

func (SubmissionReceivedData) isFeedbackData() {}

var _ FeedbackData = (*SubmissionReceivedData)(nil)

type CompilationStartedData struct{}

func (CompilationStartedData) isFeedbackData() {}

var _ FeedbackData = (*CompilationStartedData)(nil)

type CompilationFinishedData struct {
	RuntimeData RuntimeData `json:"runtime_data"`
}

func (CompilationFinishedData) isFeedbackData() {}

var _ FeedbackData = (*CompilationFinishedData)(nil)

type TestingStartedData struct{}

func (TestingStartedData) isFeedbackData() {}

var _ FeedbackData = (*TestingStartedData)(nil)

type TestStartedData struct {
	TestId int64 `json:"test_id"`
}

func (TestStartedData) isFeedbackData() {}

var _ FeedbackData = (*TestStartedData)(nil)

type TestFinishedData struct {
	TestId                int64       `json:"test_id"`
	Verdict               Status      `json:"verdict"`
	SubmissionRuntimeData RuntimeData `json:"submission_runtime_data"`
	CheckerRuntimeData    RuntimeData `json:"checker_runtime_data"`
}

func (TestFinishedData) isFeedbackData() {}

var _ FeedbackData = (*TestFinishedData)(nil)

type IncrementScoreData struct {
	Delta int `json:"delta"`
}

func (IncrementScoreData) isFeedbackData() {}

var _ FeedbackData = (*IncrementScoreData)(nil)

type FinishEvaluationData struct {
	Err error `json:"err"`
}

func (FinishEvaluationData) isFeedbackData() {}

var _ FeedbackData = (*FinishEvaluationData)(nil)
