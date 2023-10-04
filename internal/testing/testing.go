package testing

import (
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/tester/internal/database"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/messaging"
	"io"
	"log"
)

func EvaluateSubmission(request messaging.EvaluationRequest, gatherer EvalResGatherer, postgres *sqlx.DB) error {
	log.Println("Starting evaluation...")
	gatherer.StartEvaluation()
	isolateInstance := isolate.GetInstance()

	log.Println("Getting programming language...")
	language, err := getProgrammingLanguage(request, postgres)
	if err != nil {
		return err
	}
	log.Println("Got programming language:", language.FullName)

	var evalReadyFile []byte

	if language.CompileCmd == nil {
		evalReadyFile = []byte(request.Submission.SourceCode)
	} else {
		log.Println("Creating isolate box...")
		box, err := isolateInstance.NewBox()
		if err != nil {
			return err
		}
		log.Println("Created isolate box:", box.Path())

		log.Println("Adding source code to isolate box...")
		codeBytes := []byte(request.Submission.SourceCode)
		err = box.AddFile(language.CodeFilename, codeBytes)
		if err != nil {
			return err
		}
		log.Println("Added source code to isolate box")

		log.Println("Starting compilation...")
		gatherer.StartCompilation()

		log.Println("Running compilation...")
		process, err := box.Run(*language.CompileCmd, nil, nil)
		if err != nil {
			return err
		}
		log.Println("Ran compilation")

		log.Println("Collecting compilation runtime data...")
		data, err := collectProcessRuntimeData(process)
		if err != nil {
			return err
		}
		log.Println("Collected compilation runtime data")

		log.Println(
			"Compilation finished. Stdout length",
			len(data.Output.Stdout),
			"Stderr length",
			len(data.Output.Stderr),
		)
		gatherer.FinishCompilation(data)

		if data.Output.ExitCode != 0 {
			log.Println("Compilation failed with exit code:", data.Output.ExitCode)
			gatherer.FinishWithCompilationError()
			return nil
		}

		// TODO: retrieve eval ready file from isolate box
	}

	// TODO: download tests
	// select id, test_filename, task_version_id, input_text_file_id, answer_text_file_id
	// from task_version_tests

	// download to /tmp/tester, move to /var/cache/tester
	// store the tests by their respective sha values in the database

	return nil
}

func getProgrammingLanguage(request messaging.EvaluationRequest, postgres *sqlx.DB) (*database.ProgrammingLanguage, error) {
	programmingLanguageId := request.Submission.LanguageId
	programmingLanguage, err := database.SelectProgrammingLanguageById(postgres, programmingLanguageId)
	return programmingLanguage, err
}

func collectProcessRuntimeData(process *isolate.Process) (*RuntimeData, error) {

	stdout, err := io.ReadAll(process.Stdout())
	if err != nil {
		return nil, err
	}

	stderr, err := io.ReadAll(process.Stderr())
	if err != nil {
		return nil, err
	}

	metrics, err := process.Wait()
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
