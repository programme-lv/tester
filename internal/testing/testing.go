package testing

import "github.com/programme-lv/tester/internal/messaging"

func EvaluateSubmission(request messaging.EvaluationRequest, gatherer EvalResGatherer) error {
	gatherer.StartEvaluation("testerInfo", 100)
	gatherer.IncrementScore(10)
	gatherer.IncrementScore(10)
	gatherer.FinishEvaluation()
	return nil
}
