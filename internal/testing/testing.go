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

	gath.StartEvaluation()
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
		process, err := submBox.Run(req.Submission.ExecCmd, inputReadCloser, &submConstrs)
		if err != nil {
			err = fmt.Errorf("failed to run submission: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		submRunData, err := utils.CollectProcessRuntimeData(process)
		if err != nil {
			err = fmt.Errorf("failed to collect submission runtime data: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}

		gath.ReportTestSubmissionRuntimeData(int64(test.ID), submRunData)

		if submRunData.Output.ExitCode != 0 || submRunData.Output.Stderr != "" {
			// err = fmt.Errorf("submission failed to run: %v", submRunData.Output)
			gath.FinishTestWithRuntimeError(int64(test.ID))
		}
		// TODO: report time limit exceeded & memory limit exceeded

		// run checker in a box
		checkerBox, err := ii.NewBox()
		if err != nil {
			err = fmt.Errorf("failed to create isolate box: %v", err)
			gath.FinishWithInternalServerError(err)
			return err
		}
		defer checkerBox.Close()

	}

	return nil
}

func modelConstrsToIsolateConstrs(constrs models.Constraints) isolate.Constraints {
	res := isolate.DefaultConstraints()
	res.CpuTimeLimInSec = constrs.CpuTimeLimInSec
	res.MemoryLimitInKB = constrs.MemoryLimitInKB
	return res
}
