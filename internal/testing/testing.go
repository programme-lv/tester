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

const textFileCachePath = "cache/text_files"

func EvaluateSubmission(request messaging.EvaluationRequest, gatherer EvalResGatherer, postgres *sqlx.DB) error {
	log.Println("Starting evaluation...")
	gatherer.StartEvaluation()
	isolateInstance := isolate.GetInstance()

	log.Println("Selecting task version...")
	taskVersion, err := database.SelectTaskVersionById(postgres, request.TaskVersionId)
	if err != nil {
		gatherer.FinishWithInternalServerError(err)
		return err
	}
	log.Println("Selected task version:", taskVersion.ID)

	log.Println("Selecting task version checker...")
	if taskVersion.CheckerID == nil {
		err := fmt.Errorf("task version checker id is nil")
		gatherer.FinishWithInternalServerError(err)
		return err
	}
	var checker *database.TestlibChecker
	checker, err = database.SelectTestlibCheckerById(postgres, *taskVersion.CheckerID)
	if err != nil {
		gatherer.FinishWithInternalServerError(err)
		return err
	}
	log.Println("Selected task version checker:", checker.ID)

	log.Println("Getting programming language...")
	language, err := getProgrammingLanguage(request, postgres)
	if err != nil {
		gatherer.FinishWithInternalServerError(err)
		return err
	}
	log.Println("Got programming language:", language.FullName)

	var compiledSubmission []byte
	if language.CompileCmd == nil {
		log.Println("No compilation needed")
	} else {
		log.Println("Starting compilation...")
		gatherer.StartCompilation()
		var compilationRuntimeData *RuntimeData
		compiledSubmission, compilationRuntimeData, err = compileSourceCode(language, request.Submission.SourceCode)
		if err != nil {
			gatherer.FinishWithInternalServerError(err)
			return err
		}
		gatherer.FinishCompilation(compilationRuntimeData)

		if compilationRuntimeData.Output.ExitCode != 0 || compiledSubmission == nil {
			log.Println("Compilation failed with exit code:", compilationRuntimeData.Output.ExitCode)
			gatherer.FinishWithCompilationError()
			return nil
		}

		log.Println("Compilation finished successfully")
	}

	log.Println("Compiling checker...")
	var compiledChecker []byte
	var checkerCompilationData *RuntimeData
	compiledChecker, checkerCompilationData, err = compileSourceCode(language, checker.Code)
	if err != nil || compiledChecker == nil {
		log.Println("stdout:", checkerCompilationData.Output.Stdout)
		log.Println("stderr:", checkerCompilationData.Output.Stderr)
		log.Println("exit code:", checkerCompilationData.Output.ExitCode)
		gatherer.FinishWithInternalServerError(err)
		return err
	}
	log.Println("Compiled checker")

	log.Println("Selecting task version tests...")
	taskVersionTests, err := database.SelectTaskVersionTestsByTaskVersionId(postgres, request.TaskVersionId)
	if err != nil {
		gatherer.FinishWithInternalServerError(err)
		return err
	}
	log.Printf("Selected %d tests.\n", len(taskVersionTests))

	log.Println("Linking task version tests to text files...")
	// TODO: shorten this code, create a function
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

	log.Println("Downloading missing text files to cache...")
	// TODO: shorten this code, create a function
	for _, test := range taskVersionTests {
		inputTextFile, ok := testInputTextFiles[test.ID]
		if !ok {
			return fmt.Errorf("could not find input text file for test %d", test.ID)
		}
		isInputTextFileInCache, err := isTextFileInCache(inputTextFile.Sha256)
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
			err = saveTextFileToCache(inputTextFile)
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
		isAnswerTextFileInCache, err := isTextFileInCache(answerTextFile.Sha256)
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
			err = saveTextFileToCache(answerTextFile)
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
			err := box.AddFile(*language.CompiledFilename, compiledSubmission)
			if err != nil {
				gatherer.FinishWithInternalServerError(err)
				return err
			}
		} else {
			err := box.AddFile(language.CodeFilename, []byte(request.Submission.SourceCode))
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

		// TODO: add checker here
	}
	log.Println(len(compiledChecker))

	gatherer.FinishEvaluation()

	return nil
}

func compileSourceCode(language *database.ProgrammingLanguage, sourceCode string) (
	compiled []byte,
	runData *RuntimeData,
	err error,
) {
	isolateInstance := isolate.GetInstance()

	log.Println("Creating isolate box...")
	var box *isolate.Box
	box, err = isolateInstance.NewBox()
	if err != nil {
		return
	}
	log.Println("Created isolate box:", box.Path())

	log.Println("Adding source code to isolate box...")
	err = box.AddFile(language.CodeFilename, []byte(sourceCode))
	if err != nil {
		return
	}
	log.Println("Added source code to isolate box")

	log.Println("Running compilation...")
	var process *isolate.Process
	process, err = box.Run(*language.CompileCmd, nil, nil)
	if err != nil {
		return
	}
	log.Println("Ran compilation command")

	log.Println("Collecting compilation runtime data...")
	runData, err = collectProcessRuntimeData(process)
	if err != nil {
		return
	}
	log.Println("Collected compilation runtime data")

	log.Println("Compilation finished")

	if box.HasFile(*language.CompiledFilename) {
		log.Println("Retrieving compiled executable...")
		compiled, err = box.GetFile(*language.CompiledFilename)
		if err != nil {
			return
		}
		log.Println("Retrieved compiled executable")
	}

	return
}

func saveTextFileToCache(textFile *database.TextFile) error {
	err := os.MkdirAll(textFileCachePath, 0755)
	if err != nil {
		return err
	}

	fileName := textFile.Sha256
	filePath := filepath.Join(textFileCachePath, fileName)
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

func isTextFileInCache(sha256 string) (bool, error) {
	fileName := sha256
	filePath := filepath.Join(textFileCachePath, fileName)
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
