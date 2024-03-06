package testing

import (
	"github.com/programme-lv/tester/pkg/messaging"
)

func EvaluateSubmission(request messaging.EvaluationRequest, gatherer EvalResGatherer) error {
	gatherer.StartEvaluation()

	return nil
}
