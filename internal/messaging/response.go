package messaging

type EvaluationResponse struct {
	FeedbackType FeedbackType `json:"feedback_type"`
	Data         FeedbackData `json:"data"`
}
