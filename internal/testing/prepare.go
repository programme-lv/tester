package testing

import (
	"context"
	"fmt"
	"sync"

	"github.com/programme-lv/tester/internal/storage"
	"github.com/programme-lv/tester/internal/testing/compilation"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/internal/testing/utils"
	"golang.org/x/sync/errgroup"
)

func PrepareEvalRequest(req *models.EvaluationRequest, gath EvalResGatherer) (
	models.PreparedEvaluationReq, error) {

	res := models.PreparedEvaluationReq{
		Submission:  models.ExecutableFile{},
		SubmConstrs: models.Constraints{},
		Checker:     models.ExecutableFile{},
		Tests:       []models.Test{},
		Subtasks:    []models.Subtask{},
	}

	resMu := sync.Mutex{}

	res.SubmConstrs = models.Constraints{
		CpuTimeLimInSec: float64(req.Limits.CPUTimeMillis * 1000),
		MemoryLimitInKB: int64(req.Limits.MemKibibytes),
	}

	errs, _ := errgroup.WithContext(context.Background())

	errs.Go(func() error {
		tests, err := downloadTests(req.Tests)
		if err != nil {
			gath.FinishWithInternalServerError(fmt.Errorf("failed to download tests: %v", err))
			return err
		}

		resMu.Lock()
		res.Tests = tests
		resMu.Unlock()

		return nil
	})

	errs.Go(func() error {
		if req.PLanguage.CompileCmd == nil {
			resMu.Lock()
			res.Submission = models.ExecutableFile{
				Content:  []byte(req.Submission),
				Filename: req.PLanguage.CodeFilename,
				ExecCmd:  req.PLanguage.ExecCmd,
			}
			resMu.Unlock()
			return nil
		}
		gath.StartCompilation()
		cRes, runData, err := compileSubmission(req)

		if err != nil {
			err = fmt.Errorf("submission compilation failed: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		gath.FinishCompilation(runData)

		if runData.Output.ExitCode != 0 || runData.Output.Stderr != "" || cRes == nil {
			gath.FinishWithCompilationError()
			return fmt.Errorf("compilation failed: %s", runData.Output.Stderr)
		}

		resMu.Lock()
		res.Submission = *cRes
		resMu.Unlock()
		return nil
	})

	errs.Go(func() error {
		checker := req.TestlibChecker
		compiled, runData, err := compilation.CompileTestlibChecker(checker)

		if err != nil {
			err = fmt.Errorf("checker compilation failed: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		if runData.Output.ExitCode != 0 || runData.Output.Stderr != "" || compiled == nil {
			err = fmt.Errorf("checker compilation failed: %s", runData.Output.Stderr)
			gath.FinishWithInternalServerError(err)
			return err
		}

		resMu.Lock()
		res.Checker = models.ExecutableFile{
			Content:  compiled,
			Filename: "checker",
			ExecCmd:  "./checker input.txt output.txt answer.txt",
		}
		resMu.Unlock()
		return nil
	})

	err := errs.Wait()
	return res, err
}

func downloadTests(tests []models.TestRef) ([]models.Test, error) {
	pTests := make([]models.Test, 0)
	for _, rTest := range tests {
		pTest := models.Test{
			ID:        rTest.ID,
			InputSHA:  rTest.InSHA256,
			AnswerSHA: rTest.AnsSHA256,
		}

		s, err := storage.GetInstance()
		if err != nil {
			err = fmt.Errorf("failed to get storage instance: %v", err)
			return nil, err
		}

		inIs, err := s.IsTextFileInCache(rTest.InSHA256)
		if err != nil {
			err = fmt.Errorf("failed to check if input file is in cache: %v", err)
			return nil, err
		}

		if !inIs {
			if rTest.InDownlUrl == nil && rTest.InContent == nil {
				err = fmt.Errorf("input file is not in cache and no download url nor content provided")
				return nil, err
			} else if rTest.InContent != nil {
				err := s.SaveTextFileToCache([]byte(*rTest.InContent))
				if err != nil {
					err = fmt.Errorf("failed to write input file: %v", err)
					return nil, err
				}
			} else if rTest.InDownlUrl != nil {
				err := s.DownloadTextFile(*rTest.InDownlUrl)
				if err != nil {
					err = fmt.Errorf("failed to download input file: %v", err)
					return nil, err
				}
			}
			// ensure input file SHA256 is correct
			err = utils.VerifySha256(rTest.InSHA256, rTest.InSHA256)
			if err != nil {
				err = fmt.Errorf("input file SHA256 verification failed: %v", err)
				return nil, err
			}
		}

		ansIs, err := s.IsTextFileInCache(rTest.AnsSHA256)
		if err != nil {
			err = fmt.Errorf("failed to check if answer file is in cache: %v", err)
			return nil, err
		}

		if !ansIs {
			if rTest.AnsDownlUrl == nil && rTest.AnsContent == nil {
				err = fmt.Errorf("answer file is not in cache and no download url nor content provided")
				return nil, err
			} else if rTest.AnsContent != nil {
				err := s.SaveTextFileToCache([]byte(*rTest.AnsContent))
				if err != nil {
					err = fmt.Errorf("failed to write answer file: %v", err)
					return nil, err
				}
			} else if rTest.AnsDownlUrl != nil {
				err := s.DownloadTextFile(*rTest.AnsDownlUrl)
				if err != nil {
					err = fmt.Errorf("failed to download answer file: %v", err)
					return nil, err
				}
			}
			err = utils.VerifySha256(rTest.AnsSHA256, rTest.AnsSHA256)
			if err != nil {
				err = fmt.Errorf("answer file SHA256 verification failed: %v", err)
				return nil, err
			}
		}

		pTests = append(pTests, pTest)
	}
	return pTests, nil
}

func compileSubmission(req *models.EvaluationRequest) (
	*models.ExecutableFile, *models.RuntimeData, error) {

	code := req.Submission
	pLang := req.PLanguage

	if pLang.CompileCmd == nil {
		return &models.ExecutableFile{
			Content:  []byte(code),
			Filename: pLang.CodeFilename,
			ExecCmd:  pLang.ExecCmd,
		}, nil, nil
	}

	fname := pLang.CodeFilename
	cCmd := *pLang.CompileCmd
	cFname := *pLang.CompiledFilename

	compiled, runData, err := compilation.CompileSourceCode(
		code, fname, cCmd, cFname)

	if err != nil {
		return nil, nil, err
	}

	return &models.ExecutableFile{
		Content:  compiled,
		Filename: *pLang.CompiledFilename,
		ExecCmd:  pLang.ExecCmd,
	}, runData, nil
}
