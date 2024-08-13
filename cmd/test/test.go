package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/checkers"
	"github.com/programme-lv/tester/internal/filestore"
	"github.com/programme-lv/tester/internal/s3downl"
	"github.com/programme-lv/tester/internal/tester"
)

func main() {
	filestore := filestore.NewFileStore(s3downl.GetS3DownloadFunc())
	filestore.StartDownloadingInBg()
	tlibCheckers := checkers.NewTestlibCheckerCompiler()
	tester := tester.NewTester(filestore, tlibCheckers)
	jsonReq, err := os.ReadFile(filepath.Join("data", "req.json"))
	if err != nil {
		panic(fmt.Errorf("failed to read request file: %w", err))
	}
	var req internal.EvaluationRequest
	json.Unmarshal(jsonReq, &req)
	err = tester.EvaluateSubmission(stdoutGathererMock{}, req)
	fmt.Printf("Error: %v\n", err)

}

type stdoutGathererMock struct {
}

func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// FinishCompilation implements tester.EvalResGatherer.
func (s stdoutGathererMock) FinishCompilation(data *internal.RuntimeData) {
	log.Printf("Compilation finished: %s", mustMarshal(data))
}

// FinishEvaluation implements tester.EvalResGatherer.
func (s stdoutGathererMock) FinishEvaluation(errIfAny error) {
	log.Printf("Evaluation finished with error: %v", errIfAny)
}

// FinishTest implements tester.EvalResGatherer.
func (s stdoutGathererMock) FinishTest(testId int64, submission *internal.RuntimeData, checker *internal.RuntimeData) {
	log.Printf("Test %d finished: %s, %s", testId, mustMarshal(submission), mustMarshal(checker))
}

// FinishTesting implements tester.EvalResGatherer.
func (s stdoutGathererMock) FinishTesting() {
	log.Printf("Testing finished")
}

// IgnoreTest implements tester.EvalResGatherer.
func (s stdoutGathererMock) IgnoreTest(testId int64) {
	log.Printf("Test %d ignored", testId)
}

// StartCompilation implements tester.EvalResGatherer.
func (s stdoutGathererMock) StartCompilation() {
	log.Println("Compilation started")
}

// StartEvaluation implements tester.EvalResGatherer.
func (s stdoutGathererMock) StartEvaluation(systemInfo string) {
	log.Printf("Evaluation started: %s", systemInfo)
}

// StartTest implements tester.EvalResGatherer.
func (s stdoutGathererMock) StartTest(testId int64) {
	log.Printf("Test %d started", testId)
}

// StartTesting implements tester.EvalResGatherer.
func (s stdoutGathererMock) StartTesting() {
	log.Printf("Testing started")
}
