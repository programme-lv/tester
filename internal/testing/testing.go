package testing

type Submission struct {
	SourceCode string
	LanguageId string
}

type EvaluationRequest struct {
	TaskVersionId int
	Submission    Submission
}

func TestEvalRequest