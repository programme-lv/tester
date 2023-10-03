package testing

import (
	"github.com/programme-lv/tester/internal/messaging"
	"log"
)

func EvaluateSubmission(request messaging.EvaluationRequest, gatherer EvalResGatherer) error {
	gatherer.StartEvaluation()
	log.Println(request)
	return nil
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
