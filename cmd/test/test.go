package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/smithy-go/ptr"
	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/checkers"
	"github.com/programme-lv/tester/internal/filestore"
	"github.com/programme-lv/tester/internal/s3downl"
	"github.com/programme-lv/tester/internal/tester"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"), config.WithSharedConfigProfile("kp"))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	filestore := filestore.NewFileStore(s3downl.GetS3DownloadFunc())
	filestore.StartDownloadingInBg()
	tlibCheckers := checkers.NewTestlibCheckerCompiler()
	tester := tester.NewTester(filestore, tlibCheckers)

	submReqQueueUrl := "https://sqs.eu-central-1.amazonaws.com/975049886115/standard_submission_queue"

	go func() {
		sqsClient := sqs.NewFromConfig(cfg)

		for i := 1; i <= 3; i++ {
			messageBody, err := os.ReadFile(filepath.Join("data", "req.json"))
			if err != nil {
				panic(fmt.Errorf("failed to read request file: %w", err))
			}
			_, err = sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
				QueueUrl:    aws.String(submReqQueueUrl),
				MessageBody: ptr.String(string(messageBody)),
			})
			if err != nil {
				fmt.Printf("failed to send message %d, %v\n", i, err)
			} else {
				fmt.Printf("sent message %d\n", i)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	sqsClient := sqs.NewFromConfig(cfg)
	for {
		output, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(submReqQueueUrl),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     5,
		})
		if err != nil {
			fmt.Printf("failed to receive messages, %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, message := range output.Messages {
			fmt.Printf("received message: %s\n", *message.Body)

			type queueMsg struct {
				EvaluationUuid string                     `json:"evaluation_uuid"`
				Request        internal.EvaluationRequest `json:"request"`
			}
			var qMsg queueMsg
			err := json.Unmarshal([]byte(*message.Body), &qMsg)
			if err != nil {
				fmt.Printf("failed to unmarshal message, %v\n", err)
				continue
			}

			err = tester.EvaluateSubmission(NewSqsResponseQueueGatherer(), qMsg.Request)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			_, err = sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(submReqQueueUrl),
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				fmt.Printf("failed to delete message, %v\n", err)
			}
		}
	}
}

type sqsResponseQueueGatherer struct {
	sqsClient *sqs.Client
	queueUrl  string
}

// FinishCompilation implements tester.EvalResGatherer.
func (s *sqsResponseQueueGatherer) FinishCompilation(data *internal.RuntimeData) {
	msg := struct {
		MsgType     string                `json:"msg_type"`
		RuntimeData *internal.RuntimeData `json:"runtime_data"`
	}{
		MsgType:     "finished_compilation",
		RuntimeData: trimRuntimeData(data),
	}
	s.send(msg)
}

// FinishEvaluation implements tester.EvalResGatherer.
func (s *sqsResponseQueueGatherer) FinishEvaluation(errIfAny error) {
	msg := struct {
		MsgType string  `json:"msg_type"`
		Error   *string `json:"error"`
	}{
		MsgType: "finished_evaluation",
		Error:   nil,
	}
	if errIfAny != nil {
		errMsg := errIfAny.Error()
		msg.Error = &errMsg
	}
	s.send(msg)
}

// FinishTest implements tester.EvalResGatherer.
func (s *sqsResponseQueueGatherer) FinishTest(testId int64, submission *internal.RuntimeData, checker *internal.RuntimeData) {
	msg := struct {
		MsgType    string                `json:"msg_type"`
		TestId     int64                 `json:"test_id"`
		Submission *internal.RuntimeData `json:"submission"`
		Checker    *internal.RuntimeData `json:"checker"`
	}{
		MsgType:    "finished_test",
		TestId:     testId,
		Submission: trimRuntimeData(submission),
		Checker:    trimRuntimeData(checker),
	}
	s.send(msg)
}

func trimRuntimeData(data *internal.RuntimeData) *internal.RuntimeData {
	if data == nil {
		return nil
	}

	return &internal.RuntimeData{
		Stdout:          trimStringToRectangle(data.Stdout, 10, 100),
		Stderr:          trimStringToRectangle(data.Stderr, 10, 100),
		ExitCode:        data.ExitCode,
		CpuTimeMillis:   data.CpuTimeMillis,
		WallTimeMillis:  data.WallTimeMillis,
		MemoryKibiBytes: data.MemoryKibiBytes,
	}
}

func trimStringToRectangle(s *string, maxHeight int, maxWidth int) *string {
	if s == nil {
		return nil
	}
	// split into lines
	res := ""
	lines := strings.Split(*s, "\n")
	if len(lines) > maxHeight {
		lines = lines[:maxHeight]
		lines = append(lines, "...")
	}
	for i, line := range lines {
		if i > 0 {
			res += "\n"
		}
		if len(line) > maxWidth {
			res += line[:maxWidth] + "..."
		} else {
			res += line
		}
	}
	return &res
}

// FinishTesting implements tester.EvalResGatherer.
func (s *sqsResponseQueueGatherer) FinishTesting() {
	msg := struct {
		MsgType string `json:"msg_type"`
	}{
		MsgType: "finished_testing",
	}
	s.send(msg)
}

// IgnoreTest implements tester.EvalResGatherer.
func (s *sqsResponseQueueGatherer) IgnoreTest(testId int64) {
	msg := struct {
		MsgType string `json:"msg_type"`
		TestId  int64  `json:"test_id"`
	}{
		MsgType: "ignored_test",
		TestId:  testId,
	}
	s.send(msg)
}

// StartCompilation implements tester.EvalResGatherer.
func (s *sqsResponseQueueGatherer) StartCompilation() {
	msg := struct {
		MsgType string `json:"msg_type"`
	}{
		MsgType: "started_compilation",
	}
	s.send(msg)
}

// StartEvaluation implements tester.EvalResGatherer.
func (s *sqsResponseQueueGatherer) StartEvaluation(systemInfo string) {
	msg := struct {
		MsgType     string `json:"msg_type"`
		SystemInfo  string `json:"system_info"`
		StartedTime string `json:"started_time"`
	}{
		MsgType:     "started_evaluation",
		SystemInfo:  systemInfo,
		StartedTime: time.Now().Format(time.RFC3339),
	}
	s.send(msg)
}

// StartTest implements tester.EvalResGatherer.
func (s *sqsResponseQueueGatherer) StartTest(testId int64) {
	msg := struct {
		MsgType string `json:"msg_type"`
		TestId  int64  `json:"test_id"`
	}{
		MsgType: "started_test",
		TestId:  testId,
	}
	s.send(msg)
}

// StartTesting implements tester.EvalResGatherer.
func (s *sqsResponseQueueGatherer) StartTesting() {
	msg := struct {
		MsgType string `json:"msg_type"`
	}{
		MsgType: "started_testing",
	}
	s.send(msg)
}

func NewSqsResponseQueueGatherer() *sqsResponseQueueGatherer {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"), config.WithSharedConfigProfile("kp"))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	return &sqsResponseQueueGatherer{
		sqsClient: sqs.NewFromConfig(cfg),
		queueUrl:  "https://sqs.eu-central-1.amazonaws.com/975049886115/standard_subm_eval_results",
	}
}

func (s *sqsResponseQueueGatherer) send(msg interface{}) {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(fmt.Errorf("failed to marshal message: %w", err))
	}

	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueUrl),
		MessageBody: ptr.String(string(b)),
	})

	if err != nil {
		panic(fmt.Errorf("failed to send message: %w", err))
	}
}

type StdoutGathererMock struct {
}

func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// FinishCompilation implements tester.EvalResGatherer.
func (s StdoutGathererMock) FinishCompilation(data *internal.RuntimeData) {
	log.Printf("Compilation finished: %s", mustMarshal(data))
}

// FinishEvaluation implements tester.EvalResGatherer.
func (s StdoutGathererMock) FinishEvaluation(errIfAny error) {
	log.Printf("Evaluation finished with error: %v", errIfAny)
}

// FinishTest implements tester.EvalResGatherer.
func (s StdoutGathererMock) FinishTest(testId int64, submission *internal.RuntimeData, checker *internal.RuntimeData) {
	log.Printf("Test %d finished: %s, %s", testId, mustMarshal(submission), mustMarshal(checker))
}

// FinishTesting implements tester.EvalResGatherer.
func (s StdoutGathererMock) FinishTesting() {
	log.Printf("Testing finished")
}

// IgnoreTest implements tester.EvalResGatherer.
func (s StdoutGathererMock) IgnoreTest(testId int64) {
	log.Printf("Test %d ignored", testId)
}

// StartCompilation implements tester.EvalResGatherer.
func (s StdoutGathererMock) StartCompilation() {
	log.Println("Compilation started")
}

// StartEvaluation implements tester.EvalResGatherer.
func (s StdoutGathererMock) StartEvaluation(systemInfo string) {
	log.Printf("Evaluation started: %s", systemInfo)
}

// StartTest implements tester.EvalResGatherer.
func (s StdoutGathererMock) StartTest(testId int64) {
	log.Printf("Test %d started", testId)
}

// StartTesting implements tester.EvalResGatherer.
func (s StdoutGathererMock) StartTesting() {
	log.Printf("Testing started")
}
