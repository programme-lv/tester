package testing

import (
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/tester/internal/database"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/messaging"
	"io"
)

func EvaluateSubmission(request messaging.EvaluationRequest, gatherer EvalResGatherer, postgres *sqlx.DB) error {
	gatherer.StartEvaluation()

	language, err := getProgrammingLanguage(request, postgres)
	if err != nil {
		return err
	}

	isolateInstance := isolate.GetInstance()
	box, err := isolateInstance.NewBox()
	if err != nil {
		return err
	}

	codeBytes := []byte(request.Submission.SourceCode)
	err = box.AddFile(language.CodeFilename, codeBytes)
	if err != nil {
		return err
	}

	if language.CompileCmd != nil {
		gatherer.StartCompilation()
		process, err := box.Run(*language.CompileCmd, io.NopCloser(nil), nil)
		if err != nil {
			return err
		}

		data, err := collectProcessRuntimeData(process)
		if err != nil {
			return err
		}

		gatherer.FinishCompilation(data)
		if data.Output.ExitCode != 0 {
			gatherer.FinishWithCompilationError()
			return nil
		}
	}

	return nil
}

func getProgrammingLanguage(request messaging.EvaluationRequest, postgres *sqlx.DB) (*database.ProgrammingLanguage, error) {
	programmingLanguageId := request.Submission.LanguageId
	programmingLanguage, err := database.SelectProgrammingLanguageById(postgres, programmingLanguageId)
	return programmingLanguage, err
}

func collectProcessRuntimeData(process *isolate.Process) (*RuntimeData, error) {
	metrics, err := process.Wait()
	if err != nil {
		return nil, err
	}

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		return nil, err
	}

	stderr, err := io.ReadAll(process.Stderr())
	if err != nil {
		return nil, err
	}

	return &RuntimeData{
		Output: RuntimeOutput{
			Stdout:   string(stdout),
			Stderr:   string(stderr),
			ExitCode: metrics.ExitCode,
		},
		Metrics: RuntimeMetrics{
			CpuTimeMillis:  metrics.TimeSec * 1000,
			WallTimeMillis: metrics.TimeWallSec * 1000,
			MemoryKBytes:   metrics.CgMemKb,
		},
	}, nil
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
