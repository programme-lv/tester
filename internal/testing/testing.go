package testing

import (
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/tester/internal/database"
	"github.com/programme-lv/tester/internal/messaging"
	"log"
)

func EvaluateSubmission(request messaging.EvaluationRequest, gatherer EvalResGatherer, postgres *sqlx.DB) error {
	gatherer.StartEvaluation()

	programmingLanguage, err := getProgrammingLanguage(request, postgres)
	panicOnError(err)

	log.Printf("Programming language: %v\n", programmingLanguage)
	return nil
}

func getProgrammingLanguage(request messaging.EvaluationRequest, postgres *sqlx.DB) (*database.ProgrammingLanguage, error) {
	programmingLanguageId := request.Submission.LanguageId
	programmingLanguage, err := database.SelectProgrammingLanguageById(postgres, programmingLanguageId)
	return programmingLanguage, err
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
