package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/joho/godotenv"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal/filestore"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testlib"
	"github.com/programme-lv/tester/internal/xdg"
	"github.com/programme-lv/tester/sqsgath"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Try to load .env file if it exists, but don't fail if it doesn't
	err = godotenv.Load()
	if err != nil {
		log.Printf("No .env file found or error loading .env file: %v (this is optional)", err)
	}

	// Initialize XDG directories
	xdgDirs := xdg.NewXDGDirs()

	// Use XDG cache directory for file storage (persistent across restarts)
	fileDir := xdgDirs.AppCacheDir("tester/files")
	err = xdgDirs.EnsureDir(fileDir)
	if err != nil {
		log.Fatalf("failed to create file storage directory: %v", err)
	}

	// Use XDG runtime directory for temporary files (cleaned on logout/reboot)
	tmpDir := xdgDirs.AppRuntimeDir("tester")
	err = xdgDirs.EnsureRuntimeDir(tmpDir)
	if err != nil {
		log.Fatalf("failed to create tmp directory: %v", err)
	}

	filestore := filestore.New(fileDir, tmpDir)
	go filestore.Start()

	tlibCompiler := testlib.NewTestlibCompiler()

	// Read configuration assets from /usr/local/etc/tester
	configDir := "/usr/local/etc/tester"

	readFileIfExists := func(path string) (string, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", err
		}
		return string(data), nil
	}

	systemInfoTxt, err := readFileIfExists(configDir + "/system.txt")
	if err != nil {
		log.Fatalf("failed to read system.txt: %v", err)
	}
	if systemInfoTxt == "" {
		log.Printf("system.txt not found or empty in %s; proceeding with empty system info", configDir)
	}

	testlibHStr, err := readFileIfExists(configDir + "/testlib.h")
	if err != nil {
		log.Fatalf("failed to read testlib.h: %v", err)
	}
	if testlibHStr == "" {
		log.Printf("testlib.h not found or empty in %s; checker/interactor compilation may fail", configDir)
	}

	t := testing.NewTester(filestore, tlibCompiler, systemInfoTxt, testlibHStr)

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
			var request api.ExecReq
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

			log.Printf("received request with uuid: %s", request.EvalUuid)
			if request.Checker != nil {
				log.Printf("checker: %s", *request.Checker)
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
