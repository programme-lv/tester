package messaging

import "github.com/programme-lv/tester/internal/messaging/statuses"

type FeedbackType string

const (
	UpdateEvalStatus    FeedbackType = "update_eval_status"
	CompilationFinished FeedbackType = "compilation_finished"
	TestStarted         FeedbackType = "test_started"
	TestFinished        FeedbackType = "test_finished"
	SetMaxScore         FeedbackType = "set_max_score"
	IncrementScore      FeedbackType = "increment_score"
)

/*
	REPLACED BY UpdateEvalStatus:
	SubmissionReceived  FeedbackType = "submission_received"
	CompilationStarted  FeedbackType = "compilation_started"
	TestingStarted      FeedbackType = "testing_started"
	FinishEvaluation    FeedbackType = "finish_evaluation"
*/

type UpdateEvalStatusData struct {
	Status statuses.Status `json:"status"`
}

type CompilationFinishedData struct {
	RuntimeData *RuntimeData `json:"runtime_data"`
}

type TestStartedData struct {
	TestId int64 `json:"test_id"`
}

type TestFinishedData struct {
	TestId                int64           `json:"test_id"`
	Verdict               statuses.Status `json:"verdict"`
	SubmissionRuntimeData *RuntimeData    `json:"submission_runtime_data,omitempty"`
	CheckerRuntimeData    *RuntimeData    `json:"checker_runtime_data,omitempty"`
}

type SetMaxScoreData struct {
	MaxScore int `json:"max_score"`
}

type IncrementScoreData struct {
	Delta int `json:"delta"`
}

type FeedbackData interface {
	isFeedbackData()
}

func (CompilationFinishedData) isFeedbackData() {}
func (SetMaxScoreData) isFeedbackData()         {}
func (IncrementScoreData) isFeedbackData()      {}
func (TestStartedData) isFeedbackData()         {}
func (TestFinishedData) isFeedbackData()        {}
func (UpdateEvalStatusData) isFeedbackData()    {}
