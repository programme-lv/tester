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
	req internal.EvalReq,
) error {
	t.logger.Printf("Starting evaluation for submission: %s", req.Code)
	gath.StartEvaluation(t.systemInfo)

	for _, test := range req.Tests {
		// t.logger.Printf("Scheduling download for input file: %s", test.InputSha256)
		if test.InputS3Url == nil || test.AnswerS3Url == nil {
			errMsg := fmt.Errorf("input or answer S3 url is nil")
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
		err := t.filestore.ScheduleDownloadFromS3(test.InputSha256, *test.InputS3Url)
		if err != nil {
			errMsg := fmt.Errorf("failed to schedule file for download: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		// t.logger.Printf("Scheduling download for answer file: %s", test.AnswerSha256)
		err = t.filestore.ScheduleDownloadFromS3(test.AnswerSha256, *test.AnswerS3Url)
		if err != nil {
			errMsg := fmt.Errorf("failed to schedule file for download: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
	}

	// t.logger.Printf("Retrieving testlib checker: %s", req.TestlibChecker)
	tlibChecker, err := t.tlibCheckers.GetExecutable(req.Checker)
	if err != nil {
		errMsg := fmt.Errorf("failed to get testlib checker: %w", err)
		t.logger.Printf("Error: %s", errMsg)
		gath.FinishEvalWithInternalError(errMsg.Error())
		return errMsg
	}
	checkerFname := "checker"
	checkerExecCmd := "./checker input.txt output.txt answer.txt"

	var compiled []byte
	if req.Language.CompileCmd != nil {
		t.logger.Printf("Starting compilation for language: %s", req.Language.LangName)
		gath.StartCompilation()
		var runData *internal.RuntimeData

		compileBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("failed to create isolate box: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
		defer compileBox.Close()

		err = compileBox.AddFile(req.Language.CodeFname, []byte(req.Code))
		if err != nil {
			errMsg := fmt.Errorf("failed to add submission to isolate box: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		compileProcess, err := compileBox.Run(*req.Language.CompileCmd, nil, nil)
		if err != nil {
			errMsg := fmt.Errorf("failed to run compilation: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		runData, err = utils.CollectProcessRuntimeData(compileProcess)
		if err != nil {
			errMsg := fmt.Errorf("failed to collect compilation runtime data: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
		gath.FinishCompilation(runData)

		if runData.ExitCode != 0 || runData.Stderr != nil && *runData.Stderr != "" {
			errMsg := ""
			if runData.Stderr != nil {
				errMsg = fmt.Sprintf("compilation failed: %s", (*runData.Stderr)[:min(len(*runData.Stderr), 100)])
				t.logger.Printf("Error: %s", errMsg)
			} else {
				errMsg = fmt.Sprintf("compilation failed with exit code: %d", runData.ExitCode)
				t.logger.Printf("Error: %s", errMsg)
			}
			gath.FinishEvalWithCompileError(errMsg)
			return nil
		}

		if compileBox.HasFile(*req.Language.CompiledFname) {
			compiled, err = compileBox.GetFile(*req.Language.CompiledFname)
			if err != nil {
				errMsg := fmt.Errorf("failed to get compiled executable: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
		} else {
			errMsg := fmt.Errorf("compiled executable not found")
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
	}

	submFname := req.Language.CodeFname
	if compiled != nil {
		submFname = *req.Language.CompiledFname
	}

	submContent := compiled
	if submContent == nil {
		submContent = []byte(req.Code)
	}

	t.logger.Printf("Starting testing for submission: %s", req.Code)
	gath.StartTesting()
	for _, test := range req.Tests {
		t.logger.Printf("Starting test: %d", test.ID)

		t.logger.Printf("Awaiting test input: %s", test.InputSha256)
		input, err := t.filestore.AwaitAndGetFile(test.InputSha256)
		if err != nil {
			errMsg := fmt.Errorf("failed to get test input: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		t.logger.Printf("Awaiting test answer: %s", test.AnswerSha256)
		answer, err := t.filestore.AwaitAndGetFile(test.AnswerSha256)
		if err != nil {
			errMsg := fmt.Errorf("failed to get test answer: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		inputStr := string(input)
		answerStr := string(answer)
		gath.ReachTest(int64(test.ID), &inputStr, &answerStr)

		var submissionRuntimeData *internal.RuntimeData = nil
		var checkerRuntimeData *internal.RuntimeData = nil

		submBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("failed to create isolate box: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
		defer submBox.Close()

		err = submBox.AddFile(submFname, submContent)
		if err != nil {
			errMsg := fmt.Errorf("failed to add submission to isolate box: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		inputReadCloser := io.NopCloser(bytes.NewReader(input))
		submProcess, err := submBox.Run(req.Language.ExecCmd,
			inputReadCloser, &isolate.Constraints{
				CpuTimeLimInSec:      float64(req.CpuMillis) / 1000,
				ExtraCpuTimeLimInSec: 0.5,
				WallTimeLimInSec:     20.0,
				MemoryLimitInKB:      int64(req.MemoryKiB),
				MaxProcesses:         256,
				MaxOpenFiles:         256,
			})
		if err != nil {
			errMsg := fmt.Errorf("failed to run submission: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		submissionRuntimeData, err = utils.CollectProcessRuntimeData(submProcess)
		if err != nil {
			errMsg := fmt.Errorf("failed to collect submission runtime data: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		if submissionRuntimeData.ExitSignal != nil {
			errMsg := fmt.Errorf("test %d failed with signal: %d", test.ID, *submissionRuntimeData.ExitSignal)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishTest(int64(test.ID), submissionRuntimeData, nil)
			continue
		}

		if submissionRuntimeData.ExitCode != 0 ||
			submissionRuntimeData.Stderr == nil ||
			*submissionRuntimeData.Stderr != "" {
			errMsg := fmt.Errorf("test %d failed with exit code: %d", test.ID, submissionRuntimeData.ExitCode)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishTest(int64(test.ID), submissionRuntimeData, nil)
			continue
		}

		if submissionRuntimeData.WallTimeMillis > 14000 { // more than 14 seconds
			errMsg := fmt.Errorf("test %d exceeded wall time limit: %d", test.ID, submissionRuntimeData.WallTimeMillis)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishTest(int64(test.ID), submissionRuntimeData, nil)
			continue
		}

		if submissionRuntimeData.CpuTimeMillis > int64(req.CpuMillis) {
			errMsg := fmt.Errorf("test %d exceeded CPU time limit: %d", test.ID, submissionRuntimeData.CpuTimeMillis)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishTest(int64(test.ID), submissionRuntimeData, nil)
			continue
		}

		if submissionRuntimeData.MemoryKibiBytes > int64(req.MemoryKiB) {
			errMsg := fmt.Errorf("test %d exceeded memory limit: %d", test.ID, submissionRuntimeData.MemoryKibiBytes)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishTest(int64(test.ID), submissionRuntimeData, nil)
			continue
		}

		output := submissionRuntimeData.Stdout
		if output == nil {
			errMsg := fmt.Errorf("submission stdout is nil")
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		t.logger.Printf("Setting up checker for test: %d", test.ID)
		checkerBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("failed to create isolate box: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
		defer checkerBox.Close()

		err = checkerBox.AddFile(checkerFname, tlibChecker)
		if err != nil {
			errMsg := fmt.Errorf("failed to add checker to isolate box: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		err = checkerBox.AddFile("input.txt", input)
		if err != nil {
			errMsg := fmt.Errorf("failed to add input to isolate box: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		err = checkerBox.AddFile("output.txt", []byte(*output))
		if err != nil {
			errMsg := fmt.Errorf("failed to add output to isolate box: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		err = checkerBox.AddFile("answer.txt", answer)
		if err != nil {
			errMsg := fmt.Errorf("failed to add answer to isolate box: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		t.logger.Printf("Running checker for test: %d", test.ID)
		checkerProcess, err := checkerBox.Run(checkerExecCmd, nil, nil)
		if err != nil {
			errMsg := fmt.Errorf("failed to run checker: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		checkerRuntimeData, err = utils.CollectProcessRuntimeData(checkerProcess)
		if err != nil {
			errMsg := fmt.Errorf("failed to collect checker runtime data: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		t.logger.Printf("Test %d finished successfully", test.ID)
		gath.FinishTest(int64(test.ID), submissionRuntimeData, checkerRuntimeData)
	}

	t.logger.Printf("Finished testing for submission")
	gath.FinishTesting()

	t.logger.Printf("Evaluation completed for submission")
	gath.FinishEvalWithoutError()

	return nil
}
