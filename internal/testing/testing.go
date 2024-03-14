package testing

import (
	"github.com/programme-lv/tester/pkg/messaging"
)

func EvaluateSubmission(request messaging.EvaluationRequest, gath EvalResGatherer) error {
	gath.StartEvaluation()

	_, err := PrepareEvalRequest(request, gath)
	if err != nil {
		gath.FinishWithInternalServerError(err)
		return err
	}

	return nil
}
