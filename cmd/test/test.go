package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

			err = tester.EvaluateSubmission(stdoutGathererMock{}, qMsg.Request)
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
