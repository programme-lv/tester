package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/joho/godotenv"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/tester"
	"github.com/programme-lv/tester/internal/filestore"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testlib"
	"github.com/programme-lv/tester/sqsgath"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	tmpDir := filepath.Join(".", "var", "tester", "tmp")
	err = os.MkdirAll(tmpDir, 0755)
	if err != nil {
		log.Fatalf("failed to create tmp directory: %v", err)
	}

	filestore := filestore.New(filepath.Join(".", "var", "tester", "files"), tmpDir)
	go filestore.Start()

	tlibCompiler := testlib.NewTestlibCompiler()

	t := testing.NewTester(filestore, tlibCompiler)

	submReqQueueUrl := os.Getenv("SUBM_REQ_QUEUE_URL")
	if submReqQueueUrl == "" {
		log.Fatal("SUBM_REQ_QUEUE_URL environment variable is not set")
	}

	sqsClient := sqs.NewFromConfig(cfg)
	for {
		output, err := sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(submReqQueueUrl),
			MaxNumberOfMessages: 1,
			WaitTimeSeconds:     5,
		})
		if err != nil {
			log.Printf("failed to receive messages, %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, message := range output.Messages {
			log.Printf("received message: %s", *message.Body)

			// Decode base64
			compressed, err := base64.StdEncoding.DecodeString(*message.Body)
			if err != nil {
				log.Printf("failed to decode base64 message: %v", err)
				_, err = sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(submReqQueueUrl),
					ReceiptHandle: message.ReceiptHandle,
				})
				if err != nil {
					log.Printf("failed to delete message: %v", err)
				}
				continue
			}

			// Decompress zstd
			decoder, err := zstd.NewReader(nil)
			if err != nil {
				log.Printf("failed to create zstd decoder: %v", err)
				continue
			}

			jsonReq, err := decoder.DecodeAll(compressed, nil)
			if err != nil {
				log.Printf("failed to decode zstd message: %v", err)
				continue
			}
			decoder.Close()

			// Unmarshal JSON
			var request tester.EvalReq
			err = json.Unmarshal(jsonReq, &request)
			if err != nil {
				log.Printf("failed to unmarshal message: %v", err)
				_, err = sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(submReqQueueUrl),
					ReceiptHandle: message.ReceiptHandle,
				})
				if err != nil {
					log.Printf("failed to delete message: %v", err)
				}
				continue
			}

			responseSqsUrl := request.ResSqsUrl
			gatherer := sqsgath.NewSqsResponseQueueGatherer(request.EvalUuid, responseSqsUrl)
			err = t.EvaluateSubmission(gatherer, request)
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}

			_, err = sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(submReqQueueUrl),
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				log.Printf("failed to delete message: %v", err)
			}
		}
	}
}
