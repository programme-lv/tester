package testing

import (
	"context"

	"github.com/programme-lv/tester/internal/testing/compilation"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/pkg/messaging"
	"golang.org/x/sync/errgroup"
)

func PrepareEvalRequest(req messaging.EvaluationRequest, gath EvalResGatherer) (models.PreparedEvaluationReq, error) {

	res := models.PreparedEvaluationReq{
		Submission:  models.ExecutableFile{},
		SubmConstrs: models.Constraints{},
		Checker:     models.ExecutableFile{},
		Tests:       []models.TestPaths{},
		Subtasks:    []models.Subtask{},
	}

	res.SubmConstrs = models.Constraints{
		CpuTimeLimInSec: float64(req.Limits.CPUTimeMillis * 1000),
		MemoryLimitInKB: int64(req.Limits.MemKibibytes),
	}

	errs, _ := errgroup.WithContext(context.Background())

	errs.Go(func() error {
		// start downloading tests
		return nil
	})

	var resSubm models.ExecutableFile
	errs.Go(func() error {
		gath.StartCompilation()
		code := req.Submission
		pLang := req.PLanguage
		compiled, runData, err := compilation.CompileSourceCode(pLang, code)
		if err != nil {
			gath.FinishWithCompilationError()
			return err
		}
		resSubm = models.ExecutableFile{
			Content: compiled,
			ExecCmd: pLang.ExecCmd,
		}
		gath.FinishCompilation(runData)
		return nil
	})
	res.Submission = resSubm

	var resChecker models.ExecutableFile
	errs.Go(func() error {
		checker := req.TestlibChecker
		compiled, _, err := compilation.CompileTestlibChecker(checker)
		if err != nil {
			gath.FinishWithInternalServerError(err)
			return err
		}
		resChecker = models.ExecutableFile{
			Content: compiled,
			ExecCmd: "./checker input.txt output.txt answer.txt",
		}
		return nil
	})
	res.Checker = resChecker

	err := errs.Wait()
	return res, err
}
