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

// interactorConstraints are the fixed isolate limits for the jury
// interactor. They are separate from the solution's limits and are the same
// for every task.
var interactorConstraints = isolate.Constraints{
	CpuTimeLimInSec:      10,
	ExtraCpuTimeLimInSec: 0.5,
	WallTimeLimInSec:     float64(wallLimitMs) / 1000,
	MemoryLimitInKB:      1024 * 1024,
	MaxProcesses:         256,
	MaxOpenFiles:         256,
}

// solutionConstraints are the isolate limits for one solution run.
func solutionConstraints(req api.ExecReq) *isolate.Constraints {
	return &isolate.Constraints{
		CpuTimeLimInSec:      float64(req.CpuMs) / 1000,
		ExtraCpuTimeLimInSec: 0.5,
		WallTimeLimInSec:     float64(wallLimitMs) / 1000,
		MemoryLimitInKB:      int64(req.RamKiB),
		MaxProcesses:         256,
		MaxOpenFiles:         256,
	}
}

type FileFetcher interface {
	Fetch(evalUUID, sha256 string) error
}

func (t *Tester) ExecTests(gath internal.ResultGatherer, req api.ExecReq) error {
	return t.execTests(gath, req, nil)
}

func (t *Tester) ExecTestsWithFileFetcher(
	gath internal.ResultGatherer,
	req api.ExecReq,
	fetcher FileFetcher,
) error {
	return t.execTests(gath, req, fetcher)
}

func (t *Tester) execTests(gath internal.ResultGatherer, req api.ExecReq, fetcher FileFetcher) error {
	// migrated to structured logging
	l := t.logger.With("uuid", req.Uuid[0:8]+"...")
	l.Info("start job", "lang", req.Lang.LangName,
		"code_len", len(req.Code), "tests", len(req.Tests),
		"groups", len(req.Groups),
		"cpu_sec", req.CpuMs/1000, "ram_mib", req.RamKiB/1024,
		"checker", req.Checker != nil, "interactor", req.Interactor != nil)
	gath.StartJob(t.systemInfo)

	files, err := t.prefetchTests(req.Uuid, req.Tests, fetcher)
	if err != nil {
		msg := "schedule and store tests"
		l.Error(msg, "error", err)
		err = fmt.Errorf("%s: %w", msg, err)
		gath.InternalError(err.Error())
		return err
	}
	defer files.close()
	local := *t
	local.files = files
	t = &local

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
	skipper := newGroupSkip(req.Groups)
	for i, test := range req.Tests {
		testID := i + 1
		if skipper.shouldIgnore(testID) {
			l.Info("ignore test", "test_id", testID)
			gath.IgnoreTest(int64(testID))
			continue
		}
		l.Info("start test", "test_id", testID)
		err := t.runCheckerTest(gath, req, l, skipper, testID, test, submFname, submContent, tlibChecker)
		if err != nil {
			return err
		}
	}
	return nil
}

// runCheckerTest runs one checker-based test and reports its result. A
// per-test scope closes the isolate boxes as soon as the test is over.
func (t *Tester) runCheckerTest(
	gath internal.ResultGatherer,
	req api.ExecReq,
	l *slog.Logger,
	skipper *groupSkip,
	testID int,
	test api.Test,
	submFname string,
	submContent []byte,
	tlibChecker []byte,
) error {
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
	input, err := t.files.await(*test.In.Sha256)
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
	answer, err := t.files.await(*test.Ans.Sha256)
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

	submCmd, err := submBox.Command(req.Lang.ExecCmd, solutionConstraints(req))
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
		finishTrackedTest(gath, skipper, testID, submData, nil, req.CpuMs, req.RamKiB)
		return nil
	}

	if submData.ExitCode != 0 || len(submData.Stderr) > 0 {
		l.Error("submission failed", "test_id", testID, "exit_code", submData.ExitCode)
		finishTrackedTest(gath, skipper, testID, submData, nil, req.CpuMs, req.RamKiB)
		return nil
	}

	if submData.WallMillis > wallLimitMs {
		l.Error("wall time exceeded", "test_id", testID, "wall_ms", submData.WallMillis)
		finishTrackedTest(gath, skipper, testID, submData, nil, req.CpuMs, req.RamKiB)
		return nil
	}

	if submData.CpuMillis > int64(req.CpuMs) {
		l.Error("cpu time exceeded", "test_id", testID, "cpu_ms", submData.CpuMillis)
		finishTrackedTest(gath, skipper, testID, submData, nil, req.CpuMs, req.RamKiB)
		return nil
	}

	if submData.RamKiBytes > int64(req.RamKiB) {
		l.Error("memory exceeded", "test_id", testID, "mem_kib", submData.RamKiBytes)
		finishTrackedTest(gath, skipper, testID, submData, nil, req.CpuMs, req.RamKiB)
		return nil
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
	finishTrackedTest(gath, skipper, testID, submData, checkerRuntimeData, req.CpuMs, req.RamKiB)
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
	skipper := newGroupSkip(req.Groups)
	for i, test := range req.Tests {
		testID := i + 1
		if skipper.shouldIgnore(testID) {
			l.Info("ignore test", "test_id", testID)
			gath.IgnoreTest(int64(testID))
			continue
		}
		l.Info("start test", "test_id", testID)
		err := t.runInteractTest(gath, req, l, skipper, testID, test, submFname, submContent, tlibInteractor)
		if err != nil {
			return err
		}
	}
	return nil
}

// runInteractTest runs one interactive test and reports its result. A
// per-test scope closes the isolate boxes as soon as the test is over. An
// error return means a judge failure, which aborts the whole job.
func (t *Tester) runInteractTest(
	gath internal.ResultGatherer,
	req api.ExecReq,
	l *slog.Logger,
	skipper *groupSkip,
	testID int,
	test api.Test,
	submFname string,
	submContent []byte,
	tlibInteractor []byte,
) error {
	if test.In.Sha256 == nil {
		errMsg := fmt.Errorf("test %d: input sha256 is nil", testID)
		l.Error("input sha256 is nil", "test_id", testID)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	input, err := t.files.await(*test.In.Sha256)
	if err != nil {
		errMsg := fmt.Errorf("test %d: get test input: %w", testID, err)
		l.Error("get test input", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	if test.Ans.Sha256 == nil {
		errMsg := fmt.Errorf("test %d: answer sha256 is nil", testID)
		l.Error("answer sha256 is nil", "test_id", testID)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	answer, err := t.files.await(*test.Ans.Sha256)
	if err != nil {
		errMsg := fmt.Errorf("test %d: get test answer: %w", testID, err)
		l.Error("get test answer", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	gath.ReachTest(int64(testID), input, answer)

	submBox, err := isolate.NewBox()
	if err != nil {
		errMsg := fmt.Errorf("test %d: create submission box: %w", testID, err)
		l.Error("create isolate box for submission", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	defer submBox.Close()
	if err := submBox.AddFile(submFname, submContent); err != nil {
		errMsg := fmt.Errorf("test %d: add submission to box: %w", testID, err)
		l.Error("add submission to box", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}

	interactorBox, err := isolate.NewBox()
	if err != nil {
		errMsg := fmt.Errorf("test %d: create interactor box: %w", testID, err)
		l.Error("create isolate box for interactor", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	defer interactorBox.Close()
	interactorFiles := []struct {
		name    string
		content []byte
	}{
		{"interactor", tlibInteractor},
		{"input.txt", input},
		{"answer.txt", answer},
	}
	for _, file := range interactorFiles {
		if err := interactorBox.AddFile(file.name, file.content); err != nil {
			errMsg := fmt.Errorf("test %d: add %s to interactor box: %w", testID, file.name, err)
			l.Error("add interactor file to box", "test_id", testID, "file", file.name, "error", err)
			gath.InternalError(errMsg.Error())
			return errMsg
		}
	}

	interactorProcess, err := interactorBox.Command("./interactor input.txt output.txt answer.txt", &interactorConstraints)
	if err != nil {
		errMsg := fmt.Errorf("test %d: run interactor: %w", testID, err)
		l.Error("run interactor", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	submProcess, err := submBox.Command(req.Lang.ExecCmd, solutionConstraints(req))
	if err != nil {
		errMsg := fmt.Errorf("test %d: run submission: %w", testID, err)
		l.Error("run submission", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	if err := interactorProcess.Start(); err != nil {
		errMsg := fmt.Errorf("test %d: start interactor: %w", testID, err)
		l.Error("start interactor", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	if err := submProcess.Start(); err != nil {
		errMsg := fmt.Errorf("test %d: start submission: %w", testID, err)
		l.Error("start submission", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}

	submData, interactorData, err := exchange(l, interactorProcess, submProcess)
	if err != nil {
		errMsg := fmt.Errorf("test %d: %w", testID, err)
		l.Error("interactive exchange", "test_id", testID, "error", err)
		gath.InternalError(errMsg.Error())
		return errMsg
	}

	outcome, reason := classifyInteractor(interactorData)
	submFailed := submissionFailed(submData, req.CpuMs, req.RamKiB, false)
	if outcome == interactorBroken || (outcome == interactorTimedOut && !submFailed) {
		errMsg := fmt.Errorf("test %d: %s", testID, reason)
		l.Error("interactor failure", "test_id", testID, "reason", reason)
		gath.InternalError(errMsg.Error())
		return errMsg
	}
	if outcome == interactorTimedOut {
		l.Error("interactor timed out", "test_id", testID)
	}
	if outcome == interactorPartial {
		l.Info("partial interactor score", "test_id", testID, "percent", interactorScorePercent(interactorData))
	}
	l.Info("test finished", "test_id", testID)
	gath.FinishTest(int64(testID), submData, interactorData)
	if submFailed || outcome == interactorRejected {
		skipper.markFailed(testID)
	}
	return nil
}

// exchange pipes the interactor and the solution together, waits for both
// processes, and returns their runtime data. Runtime data records what each
// process read from stdin and wrote to stdout, so the site can render the
// interaction.
func exchange(l *slog.Logger, interactorProcess, submProcess *isolate.Cmd) (*api.RuntimeData, *api.RuntimeData, error) {
	interactorIn := interactorProcess.Stdin()
	interactorOut := interactorProcess.Stdout()
	interactorErr := interactorProcess.Stderr()
	submIn := submProcess.Stdin()
	submOut := submProcess.Stdout()
	submErr := submProcess.Stderr()

	var toSolution, fromSolution, solutionStderr, interactorStderr strings.Builder
	var eg errgroup.Group
	eg.Go(func() error {
		_, err := io.Copy(io.MultiWriter(submIn, &toSolution), interactorOut)
		if err != nil {
			l.Error("copy interactor->submission", "error", err)
		}
		submIn.Close()
		interactorOut.Close()
		return nil
	})
	eg.Go(func() error {
		_, err := io.Copy(io.MultiWriter(interactorIn, &fromSolution), submOut)
		if err != nil {
			l.Error("copy submission->interactor", "error", err)
		}
		interactorIn.Close()
		submOut.Close()
		return nil
	})
	eg.Go(func() error {
		_, err := io.Copy(&interactorStderr, interactorErr)
		if err != nil {
			l.Error("copy interactor stderr", "error", err)
		}
		interactorErr.Close()
		return nil
	})
	eg.Go(func() error {
		_, err := io.Copy(&solutionStderr, submErr)
		if err != nil {
			l.Error("copy submission stderr", "error", err)
		}
		submErr.Close()
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, nil, fmt.Errorf("wait for interactive exchange: %w", err)
	}

	submMetrics, err := submProcess.Wait()
	if err != nil {
		return nil, nil, fmt.Errorf("wait for submission: %w", err)
	}
	interactorMetrics, err := interactorProcess.Wait()
	if err != nil {
		return nil, nil, fmt.Errorf("wait for interactor: %w", err)
	}

	submData := &api.RuntimeData{
		Stdin:         toSolution.String(),
		Stdout:        fromSolution.String(),
		Stderr:        solutionStderr.String(),
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
	interactorData := &api.RuntimeData{
		Stdin:         fromSolution.String(),
		Stdout:        toSolution.String(),
		Stderr:        interactorStderr.String(),
		ExitCode:      interactorMetrics.ExitCode,
		CpuMillis:     interactorMetrics.CpuMillis,
		WallMillis:    interactorMetrics.WallMillis,
		RamKiBytes:    interactorMetrics.CgMemKb,
		CtxSwV:        interactorMetrics.CswVoluntary,
		CtxSwF:        interactorMetrics.CswForced,
		ExitSignal:    interactorMetrics.ExitSig,
		IsolateStatus: interactorMetrics.Status,
		IsolateMsg:    interactorMetrics.Message,
		CgOomKilled:   interactorMetrics.CgOomKilled,
	}
	return submData, interactorData, nil
}

func finishTrackedTest(
	gath internal.ResultGatherer,
	skipper *groupSkip,
	testID int,
	subm, chkr *api.RuntimeData,
	cpuMs, ramKiB int32,
) {
	gath.FinishTest(int64(testID), subm, chkr)
	if testFailed(subm, chkr, cpuMs, ramKiB) {
		skipper.markFailed(testID)
	}
}
