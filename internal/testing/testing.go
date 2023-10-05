package testing

import (
	"bytes"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/tester/internal/database"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/messaging"
	"io"
	"log"
	"os"
	"path/filepath"
)

func EvaluateSubmission(request messaging.EvaluationRequest, gatherer EvalResGatherer, postgres *sqlx.DB) error {
	log.Println("Starting evaluation...")
	gatherer.StartEvaluation()
	isolateInstance := isolate.GetInstance()

	taskVersion, err := database.SelectTaskVersionById(postgres, request.TaskVersionId)
	if err != nil {
		gatherer.FinishWithInternalServerError(err)
		return err
	}

	log.Println("Getting programming language...")
	language, err := getProgrammingLanguage(request, postgres)
	if err != nil {
		return err
	}
	log.Println("Got programming language:", language.FullName)

	var evalReadyFile []byte

	if language.CompileCmd == nil {
		log.Println("No compilation needed")
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

		log.Println("Compilation finished successfully")

		log.Println("Retrieving compiled executable...")
		evalReadyFile, err = box.GetFile(*language.CompiledFilename)
		if err != nil {
			return err
		}
		log.Println("Retrieved compiled executable")
	}

	log.Println("Selecting task version checker...")
	if taskVersion.CheckerID == nil {
		err := fmt.Errorf("task version checker id is nil")
		gatherer.FinishWithInternalServerError(err)
		return err
	}
	checker, err := database.SelectTestlibCheckerById(postgres, *taskVersion.CheckerID)
	if err != nil {
		gatherer.FinishWithInternalServerError(err)
		return err
	}
	log.Println("Selected task version checker:", checker.ID)

	log.Println("Selecting task version tests...")
	taskVersionTests, err := database.SelectTaskVersionTestsByTaskVersionId(postgres, request.TaskVersionId)
	if err != nil {
		gatherer.FinishWithInternalServerError(err)
		return err
	}
	log.Printf("Selected %d tests.\n", len(taskVersionTests))

	log.Println("Linking task version tests to text files...")
	testInputTextFiles := make(map[int64]*database.TextFileWithoutContent)
	testAnswerTextFiles := make(map[int64]*database.TextFileWithoutContent)

	for _, test := range taskVersionTests {
		inputTextFile, err := database.SelectTextFileByIdWithoutContent(postgres, test.InputTextFileID)
		if err != nil {
			gatherer.FinishWithInternalServerError(err)
			return err
		}
		testInputTextFiles[test.ID] = inputTextFile
		answerTextFile, err := database.SelectTextFileByIdWithoutContent(postgres, test.AnswerTextFileID)
		if err != nil {
			gatherer.FinishWithInternalServerError(err)
			return err
		}
		testAnswerTextFiles[test.ID] = answerTextFile
	}
	log.Println("Linked task version tests to text files")

	textFileCachePath := "cache/text_files"

	log.Println("Downloading missing text files to cache...")
	for _, test := range taskVersionTests {
		inputTextFile, ok := testInputTextFiles[test.ID]
		if !ok {
			return fmt.Errorf("could not find input text file for test %d", test.ID)
		}
		isInputTextFileInCache, err := isTextFileInCache(inputTextFile.Sha256, textFileCachePath)
		if err != nil {
			gatherer.FinishWithInternalServerError(err)
			return err
		}
		if !isInputTextFileInCache {
			log.Println("Downloading test file to cache...", test.TestFilename)
			inputTextFile, err := database.SelectTextFileById(postgres, test.InputTextFileID)
			if err != nil {
				gatherer.FinishWithInternalServerError(err)
				return err
			}
			log.Println("Saving test file to cache...", test.TestFilename)
			err = saveTextFileToCache(inputTextFile, textFileCachePath)
			if err != nil {
				gatherer.FinishWithInternalServerError(err)
				return err
			}
			log.Println("Downloaded & saved test file to cache:", test.TestFilename)
		}

		answerTextFile, ok := testAnswerTextFiles[test.ID]
		if !ok {
			return fmt.Errorf("could not find answer text file for test %d", test.ID)
		}
		isAnswerTextFileInCache, err := isTextFileInCache(answerTextFile.Sha256, textFileCachePath)
		if err != nil {
			gatherer.FinishWithInternalServerError(err)
			return err
		}
		if !isAnswerTextFileInCache {
			log.Println("Downloading answer file to cache...", test.TestFilename)
			answerTextFile, err := database.SelectTextFileById(postgres, test.AnswerTextFileID)
			if err != nil {
				return err
			}
			log.Println("Saving answer file to cache...", test.TestFilename)
			err = saveTextFileToCache(answerTextFile, textFileCachePath)
			if err != nil {
				gatherer.FinishWithInternalServerError(err)
				return err
			}
			log.Println("Saved answer file to cache:", test.TestFilename)
		}
	}
	log.Println("Downloaded missing text files to cache")

	gatherer.StartTesting(len(taskVersionTests))
	for _, test := range taskVersionTests {
		gatherer.StartTest(test.ID)
		log.Println("Starting test:", test.ID)
		// create a new box
		// place the executable in the box
		// read stdin from the input file
		// run the executable

		// collect runtime data
		// compare stdout with the answer file
		// compare runtime data with the limits

		log.Println("Creating isolate box...")
		box, err := isolateInstance.NewBox()
		if err != nil {
			gatherer.FinishWithInternalServerError(err)
			return err
		}
		log.Println("Created isolate box:", box.Path())

		log.Println("Adding executable to isolate box...")
		if language.CompiledFilename != nil {
			err := box.AddFile(*language.CompiledFilename, evalReadyFile)
			if err != nil {
				gatherer.FinishWithInternalServerError(err)
				return err
			}
		} else {
			err := box.AddFile(language.CodeFilename, evalReadyFile)
			if err != nil {
				gatherer.FinishWithInternalServerError(err)
				return err
			}
		}
		log.Println("Added executable to isolate box")

		log.Println("Creating input reader...")
		inputBytes, err := os.ReadFile(filepath.Join(textFileCachePath, testInputTextFiles[test.ID].Sha256))
		reader := bytes.NewReader(inputBytes)
		readCloser := io.NopCloser(reader)
		log.Println("Created input reader")

		log.Println("Running process...")
		process, err := box.Run(language.ExecuteCmd, readCloser, nil)
		if err != nil {
			gatherer.FinishWithInternalServerError(err)
			return err
		}
		log.Println("Ran process")

		log.Println("Collecting process runtime data...")
		data, err := collectProcessRuntimeData(process)
		if err != nil {
			gatherer.FinishWithInternalServerError(err)
			return err
		}
		log.Println("Collected process runtime data")

		log.Println("Results:", data.Output.ExitCode, data.Metrics.CpuTimeMillis, data.Metrics.MemoryKBytes)
		log.Println("Stdin:", string(inputBytes))
		log.Println("Stdout:", data.Output.Stdout)
		log.Println("Stderr:", data.Output.Stderr)
		gatherer.ReportTestSubmissionRuntimeData(test.ID, *data)

	}

	log.Println(len(evalReadyFile))

	return nil
}

func saveTextFileToCache(textFile *database.TextFile, cachePath string) error {
	err := os.MkdirAll(cachePath, 0755)
	if err != nil {
		return err
	}

	fileName := textFile.Sha256
	filePath := filepath.Join(cachePath, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	_, err = file.Write([]byte(textFile.Content))
	if err != nil {
		return err
	}
	return nil
}

func isTextFileInCache(sha256 string, cachePath string) (bool, error) {
	fileName := sha256
	filePath := filepath.Join(cachePath, fileName)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
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
