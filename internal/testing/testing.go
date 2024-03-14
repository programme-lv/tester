package testing

import (
	"bytes"
	"fmt"
	"io"

	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/storage"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/internal/testing/utils"
	"github.com/programme-lv/tester/pkg/messaging"
)

func EvaluateSubmission(rawReq messaging.EvaluationRequest, gath EvalResGatherer) error {
	gath.StartEvaluation()

	req, err := PrepareEvalRequest(rawReq, gath)
	if err != nil {
		return err
	}

	ii := isolate.GetInstance()
	s, err := storage.GetInstance()
	if err != nil {
		return err
	}

	gath.StartTesting(int64(len(req.Tests)))
	for _, test := range req.Tests {
		gath.StartTest(int64(test.ID))
		// run user submission in a box
		submBox, err := ii.NewBox()
		if err != nil {
			err = fmt.Errorf("failed to create isolate box: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}
		defer submBox.Close()

		err = submBox.AddFile(req.Submission.Filename, req.Submission.Content)
		if err != nil {
			err = fmt.Errorf("failed to add submission to isolate box: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		input, err := s.GetTextFile(test.InputSHA)
		if err != nil {
			err = fmt.Errorf("failed to get test input: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		inputReadCloser := io.NopCloser(bytes.NewReader(input))
		submConstrs := modelConstrsToIsolateConstrs(req.SubmConstrs)
		sProcess, err := submBox.Run(req.Submission.ExecCmd, inputReadCloser, &submConstrs)
		if err != nil {
			err = fmt.Errorf("failed to run submission: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		submRunData, err := utils.CollectProcessRuntimeData(sProcess)
		if err != nil {
			err = fmt.Errorf("failed to collect submission runtime data: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		gath.ReportTestSubmissionRuntimeData(int64(test.ID), submRunData)

		if submRunData.Output.ExitCode != 0 || submRunData.Output.Stderr != "" {
			// err = fmt.Errorf("submission failed to run: %v", submRunData.Output)
			gath.FinishTestWithRuntimeError(int64(test.ID))
			continue
		}

		timeLimitExceeded := float64(submRunData.Metrics.CpuTimeMillis) >= req.SubmConstrs.CpuTimeLimInSec*1000
		memoryLimitExceeded := float64(submRunData.Metrics.MemoryKBytes) >= float64(req.SubmConstrs.MemoryLimitInKB)
		wallTimeLimExceeded := float64(submRunData.Metrics.WallTimeMillis) >= submConstrs.WallTimeLimInSec*1000

		if timeLimitExceeded || memoryLimitExceeded || wallTimeLimExceeded {
			gath.FinishTestWithLimitExceeded(int64(test.ID), models.RuntimeExceededFlags{
				TimeLimitExceeded:     timeLimitExceeded,
				MemoryLimitExceeded:   memoryLimitExceeded,
				IdlenessLimitExceeded: wallTimeLimExceeded,
			})
			continue
		}

		// run checker in a box
		checkerBox, err := ii.NewBox()
		if err != nil {
			err = fmt.Errorf("failed to create isolate box: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}
		defer checkerBox.Close()

		err = checkerBox.AddFile(req.Checker.Filename, req.Checker.Content)
		if err != nil {
			err = fmt.Errorf("failed to add checker to isolate box: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		answer, err := s.GetTextFile(test.AnswerSHA)
		if err != nil {
			err = fmt.Errorf("failed to get test answer: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		err = checkerBox.AddFile("input.txt", input)
		if err != nil {
			err = fmt.Errorf("failed to add input to isolate box: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		err = checkerBox.AddFile("answer.txt", answer)
		if err != nil {
			err = fmt.Errorf("failed to add answer to isolate box: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		output := submRunData.Output.Stdout

		err = checkerBox.AddFile("output.txt", []byte(output))
		if err != nil {
			err = fmt.Errorf("failed to add output to isolate box: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		checkerConstrs := isolate.DefaultConstraints()
		cProcess, err := checkerBox.Run(req.Checker.ExecCmd, nil, &checkerConstrs)
		if err != nil {
			err = fmt.Errorf("failed to run checker: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		checkerRunData, err := utils.CollectProcessRuntimeData(cProcess)
		if err != nil {
			err = fmt.Errorf("failed to collect checker runtime data: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		gath.ReportTestCheckerRuntimeData(int64(test.ID), checkerRunData)

		if checkerRunData.Output.ExitCode == 0 {
			gath.FinishTestWithVerdictAccepted(int64(test.ID))
			gath.IncrementScore(1)
		} else if checkerRunData.Output.ExitCode == 1 ||
			checkerRunData.Output.ExitCode == 2 {
			gath.FinishTestWithVerdictWrongAnswer(int64(test.ID))
		} else {
			gath.FinishWithInternalServerError(fmt.Errorf("checker failed to run: %v",
				checkerRunData))
			return err
		}
	}
	gath.FinishEvaluation()

	return nil
}

func modelConstrsToIsolateConstrs(constrs models.Constraints) isolate.Constraints {
	res := isolate.DefaultConstraints()
	res.CpuTimeLimInSec = constrs.CpuTimeLimInSec
	res.MemoryLimitInKB = constrs.MemoryLimitInKB
	return res
}

/*
OK_EXIT_CODE = 0
WA_EXIT_CODE = 1
PE_EXIT_CODE = 2
FAIL_EXIT_CODE = 3
DIRTY_EXIT_CODE = 4 ????
POINTS_EXIT_CODE = 7
UNEXPECTED_EOF_EXIT_CODE = 8 (for interactors)
*/
