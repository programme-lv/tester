package testing

import (
	"context"
	"sync"

	"github.com/programme-lv/tester/internal/testing/compilation"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/pkg/messaging"
	"golang.org/x/sync/errgroup"
)

func PrepareEvalRequest(req messaging.EvaluationRequest, gath EvalResGatherer) (
	models.PreparedEvaluationReq, error) {

	res := models.PreparedEvaluationReq{
		Submission:  models.ExecutableFile{},
		SubmConstrs: models.Constraints{},
		Checker:     models.ExecutableFile{},
		Tests:       []models.TestPaths{},
		Subtasks:    []models.Subtask{},
	}

	resMu := sync.Mutex{}

	res.SubmConstrs = models.Constraints{
		CpuTimeLimInSec: float64(req.Limits.CPUTimeMillis * 1000),
		MemoryLimitInKB: int64(req.Limits.MemKibibytes),
	}

	errs, _ := errgroup.WithContext(context.Background())

	errs.Go(func() error {
		// start downloading tests
		return nil
	})

	errs.Go(func() error {
		cRes, err := compileSubmission(req, gath)
		if err != nil {
			return err
		}

		resMu.Lock()
		res.Submission = *cRes
		resMu.Unlock()
		return nil
	})

	var resChecker models.ExecutableFile
	errs.Go(func() error {
		checker := req.TestlibChecker
		compiled, _, err := compilation.CompileTestlibChecker(checker)
		if err != nil {
			gath.FinishWithInternalServerError(err)
			return err
		}
		resChecker = models.ExecutableFile{
			Content:  compiled,
			Filename: "checker",
			ExecCmd:  "./checker input.txt output.txt answer.txt",
		}
		return nil
	})
	res.Checker = resChecker

	err := errs.Wait()
	return res, err
}

func compileSubmission(req messaging.EvaluationRequest, gath EvalResGatherer) (
	*models.ExecutableFile, error) {

	code := req.Submission
	pLang := req.PLanguage

	if pLang.CompileCmd == nil {
		return &models.ExecutableFile{
			Content:  []byte(code),
			Filename: pLang.CodeFilename,
			ExecCmd:  pLang.ExecCmd,
		}, nil
	}

	gath.StartCompilation()

	fname := pLang.CodeFilename
	cCmd := *pLang.CompileCmd
	cFname := *pLang.CompiledFilename

	compiled, runData, err := compilation.CompileSourceCode(
		code, fname, cCmd, cFname)

	if err != nil {
		gath.FinishWithCompilationError()
		return nil, err
	}
	gath.FinishCompilation(runData)

	return &models.ExecutableFile{
		Content:  compiled,
		Filename: *pLang.CompiledFilename,
		ExecCmd:  pLang.ExecCmd,
	}, nil
}
