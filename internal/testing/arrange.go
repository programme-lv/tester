package testing

import (
	"context"

	"github.com/programme-lv/tester/internal/testing/compilation"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/pkg/messaging"
	"golang.org/x/sync/errgroup"
)

func ArrangeEvalRequest(req messaging.EvaluationRequest,
	gath EvalResGatherer) (models.ArrangedEvaluationReq, error) {

	res := models.ArrangedEvaluationReq{
		Submission:  models.ExecutableFile{},
		SubmConstrs: models.Constraints{},
		Checker:     models.ExecutableFile{},
		Tests:       []models.TestPaths{},
		Subtasks:    []models.Subtask{},
	}

	// isolateInstance := isolate.GetInstance()

	errs, _ := errgroup.WithContext(context.Background())

	errs.Go(func() error {
		// start downloading tests
		return nil
	})

	errs.Go(func() error {
		gath.StartCompilation()
		code := req.Submission
		pLang := req.PLanguage
		submission, runData, err := compilation.CompileSourceCode(pLang, code)
		if err != nil {
			gath.FinishWithCompilationError()
			return err
		}
		gath.FinishCompilation(runData)
		return nil
	})

	errs.Go(func() error {
		checker := req.TestlibChecker

		// start compiling checker
		return nil
	})

	err := errs.Wait()
	return res, err
}
