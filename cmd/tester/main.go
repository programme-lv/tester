package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/joho/godotenv"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal/behave"
	"github.com/programme-lv/tester/internal/filecache"
	"github.com/programme-lv/tester/internal/sqsgath"
	"github.com/programme-lv/tester/internal/termgath"
	testerpkg "github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testlib"
	"github.com/programme-lv/tester/internal/xdg"
	"github.com/urfave/cli/v3"
)

func main() {
	root := &cli.Command{
		Name:  "tester",
		Usage: "code execution worker",
		Commands: []*cli.Command{
			{
				Name:      "verify",
				Usage:     "Run system tests",
				ArgsUsage: "<behave.toml>",
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.NArg() < 1 {
						return fmt.Errorf("path to behave.toml is required")
					}
					return cmdVerify(c.Args().First())
				},
			},
			{
				Name:  "listen",
				Usage: "Listen for jobs",
				Commands: []*cli.Command{
					{
						Name:  "sqs",
						Usage: "Listen to AWS SQS queues",
						Action: func(ctx context.Context, c *cli.Command) error {
							cmdListenSQS()
							return nil
						},
					},
				},
			},
		},
	}
	if err := root.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func cmdListenSQS() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-central-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	t, _, _ := buildTester()
	submReqQueueUrl := mustEnv("SUBM_REQ_QUEUE_URL")
	responseQueueUrl := mustEnv("RESPONSE_QUEUE_URL")

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

			log.Printf("received request with uuid: %s", request.Uuid)
			if request.Checker != nil {
				log.Printf("checker: %s", *request.Checker)
			}

			gatherer := sqsgath.NewSqsResponseQueueGatherer(request.Uuid, responseQueueUrl)
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

func cmdVerify(path string) error {
	cases, err := behave.Parse(path)
	if err != nil {
		return err
	}
	t, _, _ := buildTester()
	g := termgath.New()
	for _, c := range cases {
		fmt.Printf("\n=== Suite: %s ===\n", c.Name)
		if err := t.EvaluateSubmission(g, c.Request); err != nil {
			return err
		}
	}
	return nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s environment variable is not set", key)
	}
	return v
}

func buildTester() (*testerpkg.Tester, string, string) {
	// Try to load .env file if it exists, but don't fail if it doesn't
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading .env file: %v (this is optional)", err)
	}

	// Initialize XDG directories
	xdgDirs := xdg.NewXDGDirs()

	// Use XDG cache directory for file storage (persistent across restarts)
	fileDir := xdgDirs.AppCacheDir("tester/files")
	if err := xdgDirs.EnsureDir(fileDir); err != nil {
		log.Fatalf("failed to create file storage directory: %v", err)
	}

	// Use XDG runtime directory for temporary files (cleaned on logout/reboot)
	tmpDir := xdgDirs.AppRuntimeDir("tester")
	if err := xdgDirs.EnsureRuntimeDir(tmpDir); err != nil {
		log.Fatalf("failed to create tmp directory: %v", err)
	}

	filestore := filecache.New(fileDir, tmpDir)
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

	t := testerpkg.NewTester(filestore, tlibCompiler, systemInfoTxt, testlibHStr)
	return t, systemInfoTxt, testlibHStr
}
