package testing

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"

	"github.com/programme-lv/tester/internal/storage"
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
			return err
		}

		resMu.Lock()
		res.Tests = tests
		resMu.Unlock()

		return nil
	})

	errs.Go(func() error {
		gath.StartCompilation()
		cRes, runData, err := compileSubmission(req)
		if exiterr, ok := err.(*exec.ExitError); ok {
			log.Println("Exit code:", exiterr.ExitCode())
			return err // TODO: handle compilation errors
		} else if err != nil {
			return err
		}
		gath.FinishCompilation(runData)

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

func downloadTests(tests []messaging.TestRef) ([]models.Test, error) {
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
			} else if rTest.InDownlUrl != nil {
				err := s.DownloadTextFile(rTest.InSHA256, *rTest.InDownlUrl)
				if err != nil {
					err = fmt.Errorf("failed to download input file: %v", err)
					return nil, err
				}
			} else if rTest.InContent != nil {
				err := s.SaveTextFileToCache(rTest.InSHA256, []byte(*rTest.InContent))
				if err != nil {
					err = fmt.Errorf("failed to write input file: %v", err)
					return nil, err
				}
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
			} else if rTest.AnsDownlUrl != nil {
				err := s.DownloadTextFile(rTest.AnsSHA256, *rTest.AnsDownlUrl)
				if err != nil {
					err = fmt.Errorf("failed to download answer file: %v", err)
					return nil, err
				}
			} else if rTest.AnsContent != nil {
				err := s.SaveTextFileToCache(rTest.AnsSHA256, []byte(*rTest.AnsContent))
				if err != nil {
					err = fmt.Errorf("failed to write answer file: %v", err)
					return nil, err
				}
			}
		}

		pTests = append(pTests, pTest)
	}
	return pTests, nil
}

func compileSubmission(req messaging.EvaluationRequest) (
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
