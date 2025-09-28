package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/klauspost/compress/zstd"
	"github.com/lmittmann/tint"
	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal/behave"
	"github.com/programme-lv/tester/internal/filecache"
	"github.com/programme-lv/tester/internal/gatherer/respbuilder"
	"github.com/programme-lv/tester/internal/gatherer/sqsgath"
	testerpkg "github.com/programme-lv/tester/internal/tester"
	"github.com/programme-lv/tester/internal/testlib"
	"github.com/programme-lv/tester/internal/xdg"
	"github.com/urfave/cli/v3"
)

func main() {
	w := os.Stderr
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))
	root := &cli.Command{
		Name:  "tester",
		Usage: "code execution worker",
		Commands: []*cli.Command{
			{
				Name:      "verify",
				Usage:     "Run system tests (see docs/behave.toml for an example)",
				ArgsUsage: "<behave.toml>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "enable verbose logs"},
					&cli.BoolFlag{Name: "no-color", Usage: "disable colorized output"},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					if c.NArg() < 1 {
						return cli.Exit("path to behave.toml is required; see --help", 1)
					}
					return cmdVerify(c.Args().First(), c.Bool("verbose"), c.Bool("no-color"))
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
		slog.Error("cli fatal error", "error", err)
		os.Exit(1)
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
			err = t.ExecTests(gatherer, request)
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

func cmdVerify(path string, verbose bool, noColor bool) error {
	cases, err := behave.Parse(path)
	if err != nil {
		return err
	}
	t, _, _ := buildTester()
	if !verbose {
		// use a no-op handler to suppress logs
		t.SetLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))
	} else {
		// default pretty logger already set; nothing to do
	}
	// Configure colors
	if noColor {
		color.NoColor = true
	}

	for _, c := range cases {
		fmt.Printf("=== Scenario: %s ===\n", c.Name)
		// Use response builder gatherer to produce a full ExecResponse
		rb := respbuilder.New(c.Request.Uuid)
		if err := t.ExecTests(rb, c.Request); err != nil {
			return err
		}
		response := rb.Response()
		if verbose {
			// Print a compact JSON of the ExecResponse for now
			b, _ := json.MarshalIndent(response, "", "  ")
			fmt.Println(string(b))
		}

		// compare the response status
		if c.Expect.Status != string(response.Status) {
			msg := fmt.Sprintf("status mismatch: expected %s, got %s", c.Expect.Status, response.Status)
			color.New(color.FgRed).Fprintln(os.Stderr, "FAIL")
			return cli.Exit(msg, 1)
		}

		// compare individual test results
		if len(c.Expect.TestResults) != len(response.TestResults) {
			msg := fmt.Sprintf("test len mismatch: expected %d, got %d", len(c.Expect.TestResults), len(response.TestResults))
			color.New(color.FgRed).Fprintln(os.Stderr, "FAIL")
			return cli.Exit(msg, 1)
		}
		for i, e := range c.Expect.TestResults {
			res := response.TestResults[i]
			verdict := "?"
			reason := ""

			if res.Subm.RamKiBytes > int64(c.Request.RamKiB) || res.Subm.CgOomKilled {
				verdict = "MLE"
				reason = fmt.Sprintf("memory usage %dKiB > %dKiB", res.Subm.RamKiBytes, c.Request.RamKiB)
			} else if res.Subm.CpuMillis > int64(c.Request.CpuMs) {
				verdict = "TLE"
				reason = fmt.Sprintf("cpu time %dms > %dms", res.Subm.CpuMillis, c.Request.CpuMs)
			} else if res.Subm.ExitCode != 0 || res.Subm.Stderr != "" || res.Subm.ExitSignal != nil {
				verdict = "RE"
				if res.Subm.ExitSignal != nil {
					reason = fmt.Sprintf("signal=%d", *res.Subm.ExitSignal)
				} else if res.Subm.Stderr != "" {
					stderr := res.Subm.Stderr
					if len(stderr) > 100 {
						stderr = stderr[:100] + "..."
					}
					reason = fmt.Sprintf("stderr=%s", stderr)
				} else {
					reason = fmt.Sprintf("exit code=%d", res.Subm.ExitCode)
				}
			} else if res.Chkr != nil && res.Chkr.ExitCode != 0 {
				verdict = "WA"
				reason = fmt.Sprintf("checker exit code: %d", res.Chkr.ExitCode)
			} else {
				verdict = "OK"
			}

			if verdict != e.Verdict {
				msg := fmt.Sprintf("test %d verdict mismatch: expected %s, got %s", i+1, e.Verdict, verdict)
				if reason != "" {
					msg += " (reason: " + reason + ")"
				}
				color.New(color.FgRed).Fprintln(os.Stderr, "FAIL")
				return cli.Exit(msg, 1)
			}
		}

		// pretty print success
		color.New(color.FgGreen).Fprintln(os.Stdout, "OK")
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
