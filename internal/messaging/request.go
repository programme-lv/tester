package messaging

type Submission struct {
	SourceCode string
	LanguageId string
}

type EvaluationRequest struct {
	TaskVersionId int64
	Submission    Submission
}

type Correlation struct {
	IsEvaluation bool  `json:"is_evaluation"`
	EvaluationId int64 `json:"evaluation_id,omitempty"`
	UnixMillis   int64 `json:"unix_millis"`
	RandomInt63  int64 `json:"random_int_63"`
}
