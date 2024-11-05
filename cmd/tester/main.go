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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/joho/godotenv"
	"github.com/klauspost/compress/zstd"
	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/filestore"
	"github.com/programme-lv/tester/internal/tester"
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

	filestore := filestore.NewFileStore(GetS3DownloadFunc())
	filestore.StartDownloadingInBg()
	tlibCheckers := testlib.NewTestlibCompiler()
	tester := tester.NewTester(filestore, tlibCheckers)

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
			var request internal.EvalReq
			err := json.Unmarshal([]byte(*message.Body), &request)
			if err != nil {
				log.Printf("failed to unmarshal message, %v", err)
				_, err = sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(submReqQueueUrl),
					ReceiptHandle: message.ReceiptHandle,
				})
				if err != nil {
					log.Printf("failed to delete message, %v", err)
				}
				continue
			}

			responseSqsUrl := request.ResSqsUrl
			gatherer := sqsgath.NewSqsResponseQueueGatherer(request.EvalUuid, responseSqsUrl)
			err = tester.EvaluateSubmission(gatherer, request)
			if err != nil {
				log.Printf("Error: %v", err)
				continue
			}

			_, err = sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(submReqQueueUrl),
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				log.Printf("failed to delete message, %v", err)
			}
		}
	}
}

// GetS3DownloadFunc returns a function that downloads a file from S3 to a local path.
// If the file is zstd compressed, it will be decompressed.
func GetS3DownloadFunc() func(s3Url string, path string) error {

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"))
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	s3Client := s3.NewFromConfig(cfg)

	return func(s3Url string, path string) error {
		u, err := url.Parse(s3Url)
		if err != nil {
			return fmt.Errorf("failed to parse s3 url %s: %w", s3Url, err)
		}

		if u.Scheme != "https" {
			return fmt.Errorf("invalid s3 url scheme: %s", u.Scheme)
		}

		// Extract bucket from host, assuming format bucket.s3.region.amazonaws.com
		hostParts := strings.Split(u.Host, ".")
		if len(hostParts) < 3 || hostParts[1] != "s3" {
			return fmt.Errorf("invalid s3 url host format: %s", u.Host)
		}
		bucket := hostParts[0]
		key := strings.TrimPrefix(u.Path, "/")

		out, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}
		defer out.Close()

		log.Printf("Downloading file %s from s3", s3Url)
		obj, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return fmt.Errorf("failed to download file %s from s3: %w (bucket: %s, key: %s)", s3Url, err, bucket, key)
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
