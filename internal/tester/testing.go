package tester

import (
	"bytes"
	"fmt"
	"io"

	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/utils"
)

func (t *Tester) EvaluateSubmission(
	gath EvalResGatherer,
	req internal.EvaluationRequest,
) error {
	gath.StartEvaluation(t.systemInfo)
	for _, test := range req.Tests {
		err := t.filestore.ScheduleDownloadFromS3(test.InputSha256, *test.InputS3Uri)
		if err != nil {
			errMsg := fmt.Errorf("failed to schedule file for download: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		err = t.filestore.ScheduleDownloadFromS3(test.AnswerSha256, *test.AnswerS3Uri)
		if err != nil {
			errMsg := fmt.Errorf("failed to schedule file for download: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}
	}

	tlibChecker, err := t.tlibCheckers.GetExecutable(req.TestlibChecker)
	if err != nil {
		errMsg := fmt.Errorf("failed to get testlib checker: %w", err)
		gath.FinishEvaluation(errMsg)
		return errMsg
	}
	checkerFname := "checker"
	checkerExecCmd := "./checker input.txt output.txt answer.txt"

	var compiled []byte
	if req.Language.CompileCommand != nil {
		gath.StartCompilation()
		var runData *internal.RuntimeData

		compileBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("failed to create isolate box: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}
		defer compileBox.Close()

		err = compileBox.AddFile(req.Language.SourceCodeFname, []byte(req.Submission))
		if err != nil {
			errMsg := fmt.Errorf("failed to add submission to isolate box: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		compileProcess, err := compileBox.Run(*req.Language.CompileCommand, nil, nil)
		if err != nil {
			errMsg := fmt.Errorf("failed to run compilation: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		runData, err = utils.CollectProcessRuntimeData(compileProcess)
		if err != nil {
			errMsg := fmt.Errorf("failed to collect compilation runtime data: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}
		gath.FinishCompilation(runData)

		if runData.ExitCode != 0 || runData.Stderr != nil && *runData.Stderr != "" {
			errMsg := fmt.Errorf("compilation failed: %s", *runData.Stderr)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		if compileBox.HasFile(*req.Language.CompiledFilename) {
			compiled, err = compileBox.GetFile(*req.Language.CompiledFilename)
			if err != nil {
				errMsg := fmt.Errorf("failed to get compiled executable: %w", err)
				gath.FinishEvaluation(errMsg)
				return errMsg
			}
		} else {
			errMsg := fmt.Errorf("compiled executable not found")
			gath.FinishEvaluation(errMsg)
			return errMsg
		}
	}

	submFname := req.Language.SourceCodeFname
	if compiled != nil {
		submFname = *req.Language.CompiledFilename
	}

	submContent := compiled
	if submContent == nil {
		submContent = []byte(req.Submission)
	}

	gath.StartTesting()
	for _, test := range req.Tests {
		gath.StartTest(test.Id)

		var submissionRuntimeData *internal.RuntimeData = nil
		var checkerRuntimeData *internal.RuntimeData = nil

		submBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("failed to create isolate box: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}
		defer submBox.Close()

		err = submBox.AddFile(submFname, submContent)
		if err != nil {
			errMsg := fmt.Errorf("failed to add submission to isolate box: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		input, err := t.filestore.AwaitAndGetFile(test.InputSha256)
		if err != nil {
			errMsg := fmt.Errorf("failed to get test input: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		inputReadCloser := io.NopCloser(bytes.NewReader(input))
		submProcess, err := submBox.Run(req.Language.ExecuteCommand,
			inputReadCloser, &isolate.Constraints{
				CpuTimeLimInSec:      float64(req.Limits.CpuTimeMillis) / 1000,
				ExtraCpuTimeLimInSec: 0.5,
				WallTimeLimInSec:     20.0,
				MemoryLimitInKB:      req.Limits.MemoryKibiBytes,
				MaxProcesses:         256,
				MaxOpenFiles:         256,
			})
		if err != nil {
			errMsg := fmt.Errorf("failed to run submission: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		submissionRuntimeData, err = utils.CollectProcessRuntimeData(submProcess)
		if err != nil {
			errMsg := fmt.Errorf("failed to collect submission runtime data: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		if submissionRuntimeData.ExitCode != 0 ||
			submissionRuntimeData.Stderr == nil ||
			*submissionRuntimeData.Stderr != "" {
			gath.FinishTest(test.Id, submissionRuntimeData, nil)
			continue
		}

		output := submissionRuntimeData.Stdout
		if output == nil {
			errMsg := fmt.Errorf("submission stdout is nil")
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		checkerBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("failed to create isolate box: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}
		defer checkerBox.Close()

		err = checkerBox.AddFile(checkerFname, tlibChecker)
		if err != nil {
			errMsg := fmt.Errorf("failed to add checker to isolate box: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		err = checkerBox.AddFile("input.txt", input)
		if err != nil {
			errMsg := fmt.Errorf("failed to add input to isolate box: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		err = checkerBox.AddFile("output.txt", []byte(*output))
		if err != nil {
			errMsg := fmt.Errorf("failed to add output to isolate box: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		answer, err := t.filestore.AwaitAndGetFile(test.AnswerSha256)
		if err != nil {
			errMsg := fmt.Errorf("failed to get test answer: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		err = checkerBox.AddFile("answer.txt", answer)
		if err != nil {
			errMsg := fmt.Errorf("failed to add answer to isolate box: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		checkerProcess, err := checkerBox.Run(checkerExecCmd, nil, nil)
		if err != nil {
			errMsg := fmt.Errorf("failed to run checker: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		checkerRuntimeData, err = utils.CollectProcessRuntimeData(checkerProcess)
		if err != nil {
			errMsg := fmt.Errorf("failed to collect checker runtime data: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		gath.FinishTest(test.Id, submissionRuntimeData, checkerRuntimeData)
	}
	fmt.Printf("Compiled: %v\n", len(compiled))
	fmt.Printf("Testlib checker: %v\n", len(tlibChecker))
	gath.FinishTesting()

	gath.FinishEvaluation(nil)

	return nil
}
