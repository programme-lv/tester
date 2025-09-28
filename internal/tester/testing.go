package tester

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/testlib"
	"github.com/programme-lv/tester/internal/utils"
	"golang.org/x/sync/errgroup"
)

var errCompileFailed = errors.New("compile failed")

func (t *Tester) ExecTests(gath internal.ResultGatherer, req api.ExecReq) error {
	// migrated to structured logging
	l := t.logger.With("uuid", req.Uuid[0:8]+"...")
	l.Info("start job", "lang", req.Lang.LangName,
		"code_len", len(req.Code), "tests", len(req.Tests),
		"cpu_sec", req.CpuMs/1000, "ram_mib", req.RamKiB/1024,
		"checker", req.Checker != nil, "interactor", req.Interactor != nil)
	gath.StartJob(t.systemInfo)

	err := t.scheduleAndStoreTests(req.Tests)
	if err != nil {
		msg := "schedule and store tests"
		l.Error(msg, "error", err)
		err = fmt.Errorf("%s: %w", msg, err)
		gath.InternalError(err.Error())
		return err
	}

	var tlibInteractor []byte
	if req.Interactor != nil {
		l.Info("compiling testlib interactor")
		var err error
		tlibInteractor, err = t.tlibCheckers.CompileInteractor(*req.Interactor, t.testlibHStr)
		if err != nil {
			msg := "get testlib interactor"
			l.Error(msg, "error", err)
			wrapped := fmt.Errorf("%s: %w", msg, err)
			gath.InternalError(wrapped.Error())
			return wrapped
		}
	} else {
		if req.Checker == nil {
			req.Checker = new(string)
			*req.Checker = testlib.DefaultChecker
		}
	}

	var tlibChecker []byte
	if req.Checker != nil {
		l.Info("compiling testlib checker")
		var err error
		tlibChecker, err = t.tlibCheckers.CompileChecker(*req.Checker, t.testlibHStr)
		if err != nil {
			msg := "get testlib checker"
			l.Error(msg, "error", err)
			wrapped := fmt.Errorf("%s: %w", msg, err)
			gath.InternalError(wrapped.Error())
			return wrapped
		}
	}

	l.Info("compiling submission")
	compiled, err := t.compileSubmission(req, gath, l)
	if err != nil {
		if errors.Is(err, errCompileFailed) {
			return nil
		}
		return err
	}

	submFname := req.Lang.CodeFname
	if compiled != nil {
		submFname = *req.Lang.CompiledFname
	}

	submContent := compiled
	if submContent == nil {
		submContent = []byte(req.Code)
	}

	l.Info("starting tests")
	if tlibChecker != nil {
		if err := t.runCheckerVariant(gath, req, l, submFname, submContent, tlibChecker); err != nil {
			return err
		}
	}
	if tlibInteractor != nil {
		if err := t.runInteractorVariant(gath, req, l, submFname, submContent, tlibInteractor); err != nil {
			return err
		}
	}

	l.Info("evaluation completed")
	gath.FinishNoError()

	return nil
}

// compileSubmission compiles the submission when a compile command is provided.
// It reports compilation start/finish to the gatherer and returns the compiled
// binary bytes. On normal compile failure, it reports the failure and returns
// errCompileFailed; on internal errors it reports an internal error and returns
// a wrapped error.
func (t *Tester) compileSubmission(req api.ExecReq, gath internal.ResultGatherer, l *slog.Logger) ([]byte, error) {
	if req.Lang.CompileCmd == nil {
		return nil, nil
	}

	l.Info("starting compilation", "lang", req.Lang.LangName)
	gath.StartCompile()

	compileBox, err := isolate.NewBox()
	if err != nil {
		errMsg := fmt.Errorf("create isolate box: %w", err)
		l.Error("create isolate box", "error", err)
		gath.InternalError(errMsg.Error())
		return nil, errMsg
	}
	defer compileBox.Close()

	if err := compileBox.AddFile(req.Lang.CodeFname, []byte(req.Code)); err != nil {
		errMsg := fmt.Errorf("add submission to isolate box: %w", err)
		l.Error("add submission to box", "error", err)
		gath.InternalError(errMsg.Error())
		return nil, errMsg
	}

	compileProcess, err := compileBox.Command(*req.Lang.CompileCmd, nil)
	if err != nil {
		errMsg := fmt.Errorf("run compilation: %w", err)
		l.Error("run compilation", "error", err)
		gath.InternalError(errMsg.Error())
		return nil, errMsg
	}

	runData, err := utils.RunIsolateCmd(compileProcess, nil)
	if err != nil {
		errMsg := fmt.Errorf("collect compilation runtime data: %w", err)
		l.Error("collect compilation runtime data", "error", err)
		gath.InternalError(errMsg.Error())
		return nil, errMsg
	}
	gath.FinishCompile(runData)

	if runData.ExitCode != 0 {
		var msg string
		if len(runData.Stderr) > 0 {
			// truncate up to 100 bytes; implement a local min to avoid import
			if len(runData.Stderr) > 100 {
				msg = fmt.Sprintf("compilation failed: %s", string(runData.Stderr[:100]))
			} else {
				msg = fmt.Sprintf("compilation failed: %s", string(runData.Stderr))
			}
		} else {
			msg = fmt.Sprintf("compilation failed with exit code: %d", runData.ExitCode)
		}
		l.Error("compilation", "exit_code", runData.ExitCode)
		gath.CompileError(msg)
		return nil, errCompileFailed
	}

	if compileBox.HasFile(*req.Lang.CompiledFname) {
		compiled, err := compileBox.GetFile(*req.Lang.CompiledFname)
		if err != nil {
			errMsg := fmt.Errorf("get compiled executable: %w", err)
			l.Error("get compiled executable", "error", err)
			gath.InternalError(errMsg.Error())
			return nil, errMsg
		}
		return compiled, nil
	}

	errMsg := fmt.Errorf("compiled executable not found")
	l.Error("compiled executable not found")
	gath.InternalError(errMsg.Error())
	return nil, errMsg
}

func (t *Tester) runCheckerVariant(
	gath internal.ResultGatherer,
	req api.ExecReq,
	l *slog.Logger,
	submFname string,
	submContent []byte,
	tlibChecker []byte,
) error {
	l.Info("running checker variant")
	for i, test := range req.Tests {
		testID := i + 1
		l.Info("start test", "test_id", testID)

		if test.In.Sha256 == nil {
			errMsg := fmt.Errorf("input sha256 is nil")
			l.Error("input sha256 is nil")
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		shaIn := *test.In.Sha256
		if len(shaIn) > 8 {
			shaIn = shaIn[:8]
		}
		l.Info("awaiting input", "sha", shaIn)
		input, err := t.filestore.Await(*test.In.Sha256)
		if err != nil {
			errMsg := fmt.Errorf("get test input: %w", err)
			l.Error("get test input", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		if test.Ans.Sha256 == nil {
			errMsg := fmt.Errorf("answer sha256 is nil")
			l.Error("answer sha256 is nil")
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		shaAns := *test.Ans.Sha256
		if len(shaAns) > 8 {
			shaAns = shaAns[:8]
		}
		l.Info("awaiting answer", "sha", shaAns)
		answer, err := t.filestore.Await(*test.Ans.Sha256)
		if err != nil {
			errMsg := fmt.Errorf("get test answer: %w", err)
			l.Error("get test answer", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		gath.ReachTest(int64(testID), input, answer)

		submBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("create isolate box: %w", err)
			l.Error("create isolate box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		defer submBox.Close()

		if err := submBox.AddFile(submFname, submContent); err != nil {
			errMsg := fmt.Errorf("add submission to isolate box: %w", err)
			l.Error("add submission to box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		submCmd, err := submBox.Command(req.Lang.ExecCmd,
			&isolate.Constraints{
				CpuTimeLimInSec:      float64(req.CpuMs) / 1000,
				ExtraCpuTimeLimInSec: 0.5,
				WallTimeLimInSec:     20.0,
				MemoryLimitInKB:      int64(req.RamKiB),
				MaxProcesses:         256,
				MaxOpenFiles:         256,
			})
		if err != nil {
			errMsg := fmt.Errorf("run submission: %w", err)
			l.Error("run submission", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		submData, err := utils.RunIsolateCmd(submCmd, input)
		if err != nil {
			errMsg := fmt.Errorf("run submission: %w", err)
			l.Error("collect submission runtime", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		if submData.ExitSignal != nil {
			l.Error("submission failed with signal", "test_id", testID, "signal", *submData.ExitSignal)
			gath.FinishTest(int64(testID), submData, nil)
			continue
		}

		if submData.ExitCode != 0 || submData.Stderr == "" || len(submData.Stderr) > 0 {
			l.Error("submission failed", "test_id", testID, "exit_code", submData.ExitCode)
			gath.FinishTest(int64(testID), submData, nil)
			continue
		}

		if submData.WallMillis > 14000 {
			l.Error("wall time exceeded", "test_id", testID, "wall_ms", submData.WallMillis)
			gath.FinishTest(int64(testID), submData, nil)
			continue
		}

		if submData.CpuMillis > int64(req.CpuMs) {
			l.Error("cpu time exceeded", "test_id", testID, "cpu_ms", submData.CpuMillis)
			gath.FinishTest(int64(testID), submData, nil)
			continue
		}

		if submData.RamKiBytes > int64(req.RamKiB) {
			l.Error("memory exceeded", "test_id", testID, "mem_kib", submData.RamKiBytes)
			gath.FinishTest(int64(testID), submData, nil)
			continue
		}

		output := submData.Stdout

		l.Info("running checker", "test_id", testID)
		checkerBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("create isolate box: %w", err)
			l.Error("create isolate box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		defer checkerBox.Close()

		if err := checkerBox.AddFile("checker", tlibChecker); err != nil {
			errMsg := fmt.Errorf("add checker to isolate box: %w", err)
			l.Error("add checker to box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		if err := checkerBox.AddFile("input.txt", input); err != nil {
			errMsg := fmt.Errorf("add input to isolate box: %w", err)
			l.Error("add input to box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		if err := checkerBox.AddFile("output.txt", []byte(output)); err != nil {
			errMsg := fmt.Errorf("add output to isolate box: %w", err)
			l.Error("add output to box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		if err := checkerBox.AddFile("answer.txt", answer); err != nil {
			errMsg := fmt.Errorf("add answer to isolate box: %w", err)
			l.Error("add answer to box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		checkerProcess, err := checkerBox.Command("./checker input.txt output.txt answer.txt", nil)
		if err != nil {
			errMsg := fmt.Errorf("run checker: %w", err)
			l.Error("run checker", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		checkerRuntimeData, err := utils.RunIsolateCmd(checkerProcess, nil)
		if err != nil {
			errMsg := fmt.Errorf("collect checker runtime data: %w", err)
			l.Error("collect checker runtime", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		l.Info("test finished", "test_id", testID)
		gath.FinishTest(int64(testID), submData, checkerRuntimeData)
	}
	return nil
}

func (t *Tester) runInteractorVariant(
	gath internal.ResultGatherer,
	req api.ExecReq,
	l *slog.Logger,
	submFname string,
	submContent []byte,
	tlibInteractor []byte,
) error {
	l.Info("running interactor variant")
	for i, test := range req.Tests {
		testID := i + 1
		l.Info("start test", "test_id", testID)

		l.Info("awaiting input", "sha", *test.In.Sha256)
		input, err := t.filestore.Await(*test.In.Sha256)
		if err != nil {
			errMsg := fmt.Errorf("get test input: %w", err)
			l.Error("get test input", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		l.Info("awaiting answer", "sha", *test.Ans.Sha256)
		answer, err := t.filestore.Await(*test.Ans.Sha256)
		if err != nil {
			errMsg := fmt.Errorf("get test answer: %w", err)
			l.Error("get test answer", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		gath.ReachTest(int64(testID), input, answer)

		l.Info("setting up isolate for submission")
		submBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("create isolate box for submission: %w", err)
			l.Error("create isolate box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		defer submBox.Close()

		if err := submBox.AddFile(submFname, submContent); err != nil {
			errMsg := fmt.Errorf("add submission to isolate box: %w", err)
			l.Error("add submission to box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		l.Info("setting up isolate for interactor")
		interactorBox, err := isolate.NewBox()
		if err != nil {
			errMsg := fmt.Errorf("create isolate box for interactor: %w", err)
			l.Error("create interactor box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		defer interactorBox.Close()

		if err := interactorBox.AddFile("interactor", tlibInteractor); err != nil {
			errMsg := fmt.Errorf("add interactor to isolate box: %w", err)
			l.Error("add interactor to box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		if err := interactorBox.AddFile("input.txt", input); err != nil {
			errMsg := fmt.Errorf("add input to isolate box: %w", err)
			l.Error("add input to box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		if err := interactorBox.AddFile("answer.txt", answer); err != nil {
			errMsg := fmt.Errorf("add answer to isolate box: %w", err)
			l.Error("add answer to box", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		interactorProcess, err := interactorBox.Command("./interactor input.txt output.txt answer.txt", nil)
		if err != nil {
			errMsg := fmt.Errorf("run interactor: %w", err)
			l.Error("run interactor", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		submConstraints := &isolate.Constraints{
			CpuTimeLimInSec:      float64(req.CpuMs) / 1000,
			ExtraCpuTimeLimInSec: 0.5,
			WallTimeLimInSec:     20.0,
			MemoryLimitInKB:      int64(req.RamKiB),
			MaxProcesses:         256,
			MaxOpenFiles:         256,
		}
		submProcess, err := submBox.Command(req.Lang.ExecCmd, submConstraints)
		if err != nil {
			errMsg := fmt.Errorf("run submission: %w", err)
			l.Error("run submission", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		if err := interactorProcess.Start(); err != nil {
			errMsg := fmt.Errorf("start interactor: %w", err)
			l.Error("start interactor", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		if err := submProcess.Start(); err != nil {
			errMsg := fmt.Errorf("start submission: %w", err)
			l.Error("start submission", "error", err)
			gath.InternalError(errMsg.Error())
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
				l.Error("copy interactor->submission", "error", err)
			}
			submStdin.Close()
			interactorStdout.Close()
			return nil
		})
		// move stdout from submission to stdin of interactor
		eg.Go(func() error {
			_, err := io.Copy(io.MultiWriter(interactorStdin, &submStdoutStr), submStdout)
			if err != nil {
				l.Error("copy submission->interactor", "error", err)
			}
			submStdout.Close()
			interactorStdin.Close()
			return nil
		})
		// read stderr from interactor
		eg.Go(func() error {
			_, err := io.Copy(&interactorStderrStr, interactorStderr)
			if err != nil {
				l.Error("copy interactor stderr", "error", err)
			}
			interactorStderr.Close()
			return nil
		})
		// read stderr from submission
		eg.Go(func() error {
			_, err := io.Copy(&submStderrStr, submStderr)
			if err != nil {
				l.Error("copy submission stderr", "error", err)
			}
			submStderr.Close()
			return nil
		})

		if err := eg.Wait(); err != nil {
			errMsg := fmt.Errorf("wait for interactor and submission: %w", err)
			l.Error("wait for processes", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		submMetrics, err := submProcess.Wait()
		if err != nil {
			errMsg := fmt.Errorf("wait for submission: %w", err)
			l.Error("wait for submission", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
		submissionRuntimeData := &api.RuntimeData{
			Stdin:         submStdinStr.String(),
			Stdout:        submStdoutStr.String(),
			Stderr:        submStderrStr.String(),
			ExitCode:      submMetrics.ExitCode,
			CpuMillis:     submMetrics.CpuMillis,
			WallMillis:    submMetrics.WallMillis,
			RamKiBytes:    submMetrics.CgMemKb,
			CtxSwV:        submMetrics.CswVoluntary,
			CtxSwF:        submMetrics.CswForced,
			ExitSignal:    submMetrics.ExitSig,
			IsolateStatus: submMetrics.Status,
			IsolateMsg:    submMetrics.Message,
			CgOomKilled:   submMetrics.CgOomKilled,
		}

		interactorMetrics, err := interactorProcess.Wait()
		if err != nil {
			errMsg := fmt.Errorf("wait for interactor: %w", err)
			l.Error("wait for interactor", "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}

		interactorRuntimeData := &api.RuntimeData{
			Stdin:         submStdinStr.String(),
			Stdout:        submStdinStr.String(),
			Stderr:        interactorStderrStr.String(),
			ExitCode:      interactorMetrics.ExitCode,
			CpuMillis:     interactorMetrics.CpuMillis,
			WallMillis:    interactorMetrics.WallMillis,
			RamKiBytes:    interactorMetrics.CgMemKb,
			IsolateStatus: interactorMetrics.Status,
			CtxSwV:        interactorMetrics.CswVoluntary,
			CtxSwF:        interactorMetrics.CswForced,
			ExitSignal:    interactorMetrics.ExitSig,
			IsolateMsg:    interactorMetrics.Message,
			CgOomKilled:   interactorMetrics.CgOomKilled,
		}

		l.Info("test finished", "test_id", testID)
		gath.FinishTest(int64(testID), submissionRuntimeData, interactorRuntimeData)
	}
	return nil
}

func (t *Tester) scheduleAndStoreTests(tests []api.Test) error {
	for i := range tests {
		test := &tests[i]
		if (test.In.Url == nil && test.In.Content == nil) || (test.Ans.Url == nil && test.Ans.Content == nil) {
			return errors.New("input or answer download url and content are nil")
		}
		if test.In.Content != nil {
			var err error
			if test.In.Sha256 == nil {
				test.In.Sha256 = new(string)
			}
			*test.In.Sha256, err = t.filestore.Store([]byte(*test.In.Content))
			if err != nil {
				return fmt.Errorf("store input content: %w", err)
			}
		} else {
			if test.In.Sha256 == nil {
				return errors.New("input sha256 is nil")
			}
			err := t.filestore.Schedule(*test.In.Sha256, *test.In.Url)
			if err != nil {
				return fmt.Errorf("schedule input file for download: %w", err)
			}
		}
		if test.Ans.Content != nil {
			var err error
			if test.Ans.Sha256 == nil {
				test.Ans.Sha256 = new(string)
			}
			*test.Ans.Sha256, err = t.filestore.Store([]byte(*test.Ans.Content))
			if err != nil {
				return fmt.Errorf("store answer content: %w", err)
			}
		} else {
			if test.Ans.Sha256 == nil {
				return errors.New("answer sha256 is nil")
			}
			err := t.filestore.Schedule(*test.Ans.Sha256, *test.Ans.Url)
			if err != nil {
				return fmt.Errorf("schedule answer file for download: %w", err)
			}
		}
	}

	return nil
}
