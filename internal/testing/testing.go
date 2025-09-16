package testing

import (
	"fmt"
	"io"
	"strings"

	"github.com/programme-lv/tester"
	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/testlib"
	"github.com/programme-lv/tester/internal/utils"
	"golang.org/x/sync/errgroup"
)

func (t *Tester) EvaluateSubmission(
	gath EvalResGatherer,
	req tester.EvalReq,
) error {
	t.logger.Printf("Starting evaluation for submission: %s", req.Code)
	gath.StartEvaluation(t.systemInfo)

	for i := range req.Tests {
		test := &req.Tests[i]
		if (test.InUrl == nil && test.InContent == nil) || (test.AnsUrl == nil && test.AnsContent == nil) {
			errMsg := fmt.Errorf("input or answer download url is nil and content is nil")
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
		if test.InContent != nil {
			var err error
			if test.InSha256 == nil {
				test.InSha256 = new(string)
			}
			*test.InSha256, err = t.filestore.Store([]byte(*test.InContent))
			if err != nil {
				errMsg := fmt.Errorf("failed to store input content: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
		} else {
			if test.InSha256 == nil {
				errMsg := fmt.Errorf("input sha256 is nil")
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
			err := t.filestore.Schedule(*test.InSha256, *test.InUrl)
			if err != nil {
				errMsg := fmt.Errorf("failed to schedule file for download: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
		}
		if test.AnsContent != nil {
			var err error
			if test.AnsSha256 == nil {
				test.AnsSha256 = new(string)
			}
			*test.AnsSha256, err = t.filestore.Store([]byte(*test.AnsContent))
			if err != nil {
				errMsg := fmt.Errorf("failed to store answer content: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
		} else {
			if test.AnsSha256 == nil {
				errMsg := fmt.Errorf("answer sha256 is nil")
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
			err := t.filestore.Schedule(*test.AnsSha256, *test.AnsUrl)
			if err != nil {
				errMsg := fmt.Errorf("failed to schedule file for download: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
		}
	}

	var tlibInteractor []byte
	if req.Interactor != nil {
		var err error
		tlibInteractor, err = t.tlibCheckers.CompileInteractor(*req.Interactor, t.testlibHStr)
		if err != nil {
			errMsg := fmt.Errorf("failed to get testlib interactor: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
	} else {
		if req.Checker == nil {
			req.Checker = new(string)
			*req.Checker = testlib.DefaultChecker
		}
	}

	var tlibChecker []byte
	if req.Checker != nil {
		var err error
		tlibChecker, err = t.tlibCheckers.CompileChecker(*req.Checker, t.testlibHStr)
		if err != nil {
			errMsg := fmt.Errorf("failed to get testlib checker: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
	}

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

		compileProcess, err := compileBox.Command(*req.Language.CompileCmd, nil)
		if err != nil {
			errMsg := fmt.Errorf("failed to run compilation: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}

		runData, err = utils.RunIsolateCmd(compileProcess, nil)
		if err != nil {
			errMsg := fmt.Errorf("failed to collect compilation runtime data: %w", err)
			t.logger.Printf("Error: %s", errMsg)
			gath.FinishEvalWithInternalError(errMsg.Error())
			return errMsg
		}
		gath.FinishCompilation(runData)

		if runData.ExitCode != 0 {
			errMsg := ""
			if len(runData.Stderr) > 0 {
				stderr := string(runData.Stderr[:min(len(runData.Stderr), 100)])
				errMsg = fmt.Sprintf("compilation failed: %s", stderr)
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
	if tlibChecker != nil {
		t.logger.Printf("Running checker variant")
		for _, test := range req.Tests {
			t.logger.Printf("Starting test: %d", test.ID)

			if test.InSha256 == nil {
				errMsg := fmt.Errorf("input sha256 is nil")
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
			t.logger.Printf("Awaiting test input: %s", (*test.InSha256)[:8])
			input, err := t.filestore.Await(*test.InSha256)
			if err != nil {
				errMsg := fmt.Errorf("failed to get test input: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			if test.AnsSha256 == nil {
				errMsg := fmt.Errorf("answer sha256 is nil")
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
			t.logger.Printf("Awaiting test answer: %s", (*test.AnsSha256)[:8])
			answer, err := t.filestore.Await(*test.AnsSha256)
			if err != nil {
				errMsg := fmt.Errorf("failed to get test answer: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			gath.ReachTest(int64(test.ID), input, answer)

			var submData *internal.RuntimeData = nil
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

			// inputReadCloser := io.NopCloser(bytes.NewReader(input))
			submCmd, err := submBox.Command(req.Language.ExecCmd,
				&isolate.Constraints{
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

			submData, err = utils.RunIsolateCmd(submCmd, input)
			if err != nil {
				errMsg := fmt.Errorf("failed to run submission: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			if submData.ExitSignal != nil {
				errMsg := fmt.Errorf("test %d failed with signal: %d", test.ID, *submData.ExitSignal)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishTest(int64(test.ID), submData, nil)
				continue
			}

			if submData.ExitCode != 0 ||
				submData.Stderr == nil ||
				len(submData.Stderr) > 0 {
				errMsg := fmt.Errorf("test %d failed with exit code: %d", test.ID, submData.ExitCode)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishTest(int64(test.ID), submData, nil)
				continue
			}

			if submData.WallMs > 14000 { // more than 14 seconds
				errMsg := fmt.Errorf("test %d exceeded wall time limit: %d", test.ID, submData.WallMs)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishTest(int64(test.ID), submData, nil)
				continue
			}

			if submData.CpuMs > int64(req.CpuMillis) {
				errMsg := fmt.Errorf("test %d exceeded CPU time limit: %d", test.ID, submData.CpuMs)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishTest(int64(test.ID), submData, nil)
				continue
			}

			if submData.MemKiB > int64(req.MemoryKiB) {
				errMsg := fmt.Errorf("test %d exceeded memory limit: %d", test.ID, submData.MemKiB)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishTest(int64(test.ID), submData, nil)
				continue
			}

			output := submData.Stdout
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

			err = checkerBox.AddFile("checker", tlibChecker)
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

			err = checkerBox.AddFile("output.txt", output)
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
			checkerProcess, err := checkerBox.Command("./checker input.txt output.txt answer.txt", nil)
			if err != nil {
				errMsg := fmt.Errorf("failed to run checker: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			checkerRuntimeData, err = utils.RunIsolateCmd(checkerProcess, nil)
			if err != nil {
				errMsg := fmt.Errorf("failed to collect checker runtime data: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			t.logger.Printf("Test %d finished successfully", test.ID)
			gath.FinishTest(int64(test.ID), submData, checkerRuntimeData)
		}
	}
	if tlibInteractor != nil {
		t.logger.Printf("Running interactor variant")
		for _, test := range req.Tests {
			t.logger.Printf("Starting test: %d", test.ID)

			t.logger.Printf("Awaiting test input: %s", *test.InSha256)
			input, err := t.filestore.Await(*test.InSha256)
			if err != nil {
				errMsg := fmt.Errorf("failed to get test input: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			t.logger.Printf("Awaiting test answer: %s", *test.AnsSha256)
			answer, err := t.filestore.Await(*test.AnsSha256)
			if err != nil {
				errMsg := fmt.Errorf("failed to get test answer: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			gath.ReachTest(int64(test.ID), input, answer)

			var submissionRuntimeData *internal.RuntimeData = nil
			var interactorRuntimeData *internal.RuntimeData = nil

			t.logger.Printf("Setting up isolate box for submission")
			submBox, err := isolate.NewBox()
			if err != nil {
				errMsg := fmt.Errorf("failed to create isolate box for submission: %w", err)
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

			t.logger.Printf("Setting up isolate box for interactor")
			interactorBox, err := isolate.NewBox()
			if err != nil {
				errMsg := fmt.Errorf("failed to create isolate box for interactor: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
			defer interactorBox.Close()

			err = interactorBox.AddFile("interactor", tlibInteractor)
			if err != nil {
				errMsg := fmt.Errorf("failed to add interactor to isolate box: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			err = interactorBox.AddFile("input.txt", input)
			if err != nil {
				errMsg := fmt.Errorf("failed to add input to isolate box: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			err = interactorBox.AddFile("answer.txt", answer)
			if err != nil {
				errMsg := fmt.Errorf("failed to add answer to isolate box: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			interactorProcess, err := interactorBox.Command("./interactor input.txt output.txt answer.txt", nil)
			if err != nil {
				errMsg := fmt.Errorf("failed to run interactor: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			submConstraints := &isolate.Constraints{
				CpuTimeLimInSec:      float64(req.CpuMillis) / 1000,
				ExtraCpuTimeLimInSec: 0.5,
				WallTimeLimInSec:     20.0,
				MemoryLimitInKB:      int64(req.MemoryKiB),
				MaxProcesses:         256,
				MaxOpenFiles:         256,
			}
			submProcess, err := submBox.Command(req.Language.ExecCmd, submConstraints)
			if err != nil {
				errMsg := fmt.Errorf("failed to run submission: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			err = interactorProcess.Start()
			if err != nil {
				errMsg := fmt.Errorf("failed to start interactor: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			err = submProcess.Start()
			if err != nil {
				errMsg := fmt.Errorf("failed to start submission: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			interactorStdin := interactorProcess.Stdin()
			interactorStdout := interactorProcess.Stdout()
			interactorStderr := interactorProcess.Stderr()

			submStdin := submProcess.Stdin()
			submStdout := submProcess.Stdout()
			submStderr := submProcess.Stderr()

			var submStdinStr, submStdoutStr strings.Builder
			var submStderrStr, interactorStderrStr strings.Builder

			var eg errgroup.Group
			// move stdout from interactor to stdin of submission
			eg.Go(func() error {
				_, err := io.Copy(io.MultiWriter(submStdin, &submStdinStr), interactorStdout)
				if err != nil {
					t.logger.Printf("Error copying interactor stdout to submission stdin: %v", err)
				}
				submStdin.Close()
				interactorStdout.Close()
				return nil
			})
			// move stdout from submission to stdin of interactor
			eg.Go(func() error {
				_, err := io.Copy(io.MultiWriter(interactorStdin, &submStdoutStr), submStdout)
				if err != nil {
					t.logger.Printf("Error copying submission stdout to interactor stdin: %v", err)
				}
				submStdout.Close()
				interactorStdin.Close()
				return nil
			})
			// read stderr from interactor
			eg.Go(func() error {
				_, err := io.Copy(&interactorStderrStr, interactorStderr)
				if err != nil {
					t.logger.Printf("Error copying interactor stderr to string: %v", err)
				}
				interactorStderr.Close()
				return nil
			})
			// read stderr from submission
			eg.Go(func() error {
				_, err := io.Copy(&submStderrStr, submStderr)
				if err != nil {
					t.logger.Printf("Error copying submission stderr to string: %v", err)
				}
				submStderr.Close()
				return nil
			})

			err = eg.Wait()
			if err != nil {
				errMsg := fmt.Errorf("failed to wait for interactor and submission: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			submMetrics, err := submProcess.Wait()
			if err != nil {
				errMsg := fmt.Errorf("failed to wait for submission: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}
			submissionRuntimeData = &internal.RuntimeData{
				Stdin:         []byte(submStdinStr.String()),
				Stdout:        []byte(submStdoutStr.String()),
				Stderr:        []byte(submStderrStr.String()),
				ExitCode:      submMetrics.ExitCode,
				CpuMs:         submMetrics.CpuMillis,
				WallMs:        submMetrics.WallMillis,
				MemKiB:        submMetrics.CgMemKb,
				CtxSwV:        submMetrics.CswVoluntary,
				CtxSwF:        submMetrics.CswForced,
				ExitSignal:    submMetrics.ExitSig,
				IsolateStatus: submMetrics.Status,
				IsolateMsg:    submMetrics.Message,
			}

			interactorMetrics, err := interactorProcess.Wait()
			if err != nil {
				errMsg := fmt.Errorf("failed to wait for interactor: %w", err)
				t.logger.Printf("Error: %s", errMsg)
				gath.FinishEvalWithInternalError(errMsg.Error())
				return errMsg
			}

			interactorRuntimeData = &internal.RuntimeData{
				Stdout:        []byte(submStdinStr.String()),
				Stderr:        []byte(interactorStderrStr.String()),
				ExitCode:      interactorMetrics.ExitCode,
				CpuMs:         interactorMetrics.CpuMillis,
				WallMs:        interactorMetrics.WallMillis,
				MemKiB:        interactorMetrics.CgMemKb,
				Stdin:         []byte(submStdinStr.String()),
				IsolateStatus: interactorMetrics.Status,
				CtxSwV:        interactorMetrics.CswVoluntary,
				CtxSwF:        interactorMetrics.CswForced,
				ExitSignal:    interactorMetrics.ExitSig,
				IsolateMsg:    interactorMetrics.Message,
			}

			t.logger.Printf("Test %d finished", test.ID)
			gath.FinishTest(int64(test.ID), submissionRuntimeData, interactorRuntimeData)
		}
	}

	t.logger.Printf("Finished testing for submission")
	gath.FinishTesting()

	t.logger.Printf("Evaluation completed for submission")
	gath.FinishEvalWithoutError()

	return nil
}
