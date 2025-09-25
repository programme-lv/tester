package utils

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
	"golang.org/x/sync/errgroup"
)

func RunIsolateCmd(p *isolate.Cmd, input []byte) (*internal.RuntimeData, error) {
	var eg errgroup.Group

	log.Printf("Starting isolate command")
	err := p.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start isolate command: %w", err)
	}

	// write everything to stdin
	if input != nil {
		log.Printf("Starting stdin copy goroutine")
		eg.Go(func() error {
			_, err := io.Copy(p.Stdin(), bytes.NewReader(input))
			if err != nil {
				log.Printf("failed to write to stdin: %v", err)
			}
			p.Stdin().Close()
			log.Printf("Finished stdin copy")
			return nil
		})
	}

	// read everything from stdout
	var stdout []byte
	log.Printf("Starting stdout read goroutine")
	eg.Go(func() error {
		stdout, err = io.ReadAll(p.Stdout())
		if err != nil {
			log.Printf("failed to read stdout: %v", err)
		}
		log.Printf("Finished stdout read")
		return nil
	})

	// read everything from stderr
	var stderr []byte
	log.Printf("Starting stderr read goroutine")
	eg.Go(func() error {
		stderr, err = io.ReadAll(p.Stderr())
		if err != nil {
			log.Printf("failed to read stderr: %v", err)
		}
		log.Printf("Finished stderr read")
		return nil
	})

	log.Printf("Waiting for isolate command to finish")
	metrics, err := p.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for isolate command: %w", err)
	}
	log.Printf("Isolate command finished")

	log.Printf("Waiting for all goroutines to complete")
	err = eg.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for isolate command: %w", err)
	}
	log.Printf("All goroutines completed")

	return &internal.RuntimeData{
		Stdin:         input,
		Stdout:        stdout,
		Stderr:        stderr,
		ExitCode:      metrics.ExitCode,
		CpuMs:         metrics.CpuMillis,
		WallMs:        metrics.WallMillis,
		MemKiB:        metrics.CgMemKb,
		IsolateStatus: metrics.Status,
		CtxSwV:        metrics.CswVoluntary,
		CtxSwF:        metrics.CswForced,
		ExitSignal:    metrics.ExitSig,
		IsolateMsg:    metrics.Message,
		FullReport:    metrics.FullReport,
	}, nil
}
