package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/checkers"
	"github.com/programme-lv/tester/internal/filestore"
	"github.com/programme-lv/tester/internal/tester"
)

func main() {
	filestore := filestore.NewFileStore(getS3DownloadFunc())
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

func getS3DownloadFunc() func(s3Uri string, path string) error {

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithSharedConfigProfile("kp"),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	s3Client := s3.NewFromConfig(cfg)

	return func(s3Uri string, path string) error {
		u, err := url.Parse(s3Uri)
		if err != nil {
			return fmt.Errorf("failed to parse s3 uri %s: %w", s3Uri, err)
		}

		if u.Scheme != "s3" {
			return fmt.Errorf("invalid s3 uri scheme: %s", u.Scheme)
		}

		out, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}
		defer out.Close()

		log.Printf("Downloading file %s from s3", s3Uri)
		obj, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(u.Host),
			Key:    aws.String(u.Path[1:]),
		})
		if err != nil {
			return fmt.Errorf("failed to download file %s from s3: %w (host: %s, path: %s)", s3Uri, err, u.Host, u.Path)
		}

		if (obj.ContentType != nil && *obj.ContentType == "application/zstd") ||
			filepath.Ext(u.Path) == ".zst" {

			d, err := zstd.NewReader(obj.Body)
			if err != nil {
				return fmt.Errorf("failed to create zstd reader: %w", err)
			}
			defer d.Close()
			_, err = io.Copy(out, d)
			if err != nil {
				return fmt.Errorf("failed to write file %s: %w", path, err)
			}

			return nil
		}

		_, err = io.Copy(out, obj.Body)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}

		return nil
	}

}

type stdoutGathererMock struct {
}

// FinishCompilation implements tester.EvalResGatherer.
func (s stdoutGathererMock) FinishCompilation(data *internal.RuntimeData) {
	log.Printf("Compilation finished: %+v", data)
}

// FinishEvaluation implements tester.EvalResGatherer.
func (s stdoutGathererMock) FinishEvaluation(errIfAny error) {
	log.Printf("Evaluation finished with error: %v", errIfAny)
}

// FinishTest implements tester.EvalResGatherer.
func (s stdoutGathererMock) FinishTest(testId int64, submission *internal.RuntimeData, checker *internal.RuntimeData) {
	log.Printf("Test %d finished: %+v, %+v", testId, submission, checker)
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
