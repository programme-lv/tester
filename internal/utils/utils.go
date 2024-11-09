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

	err := p.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start isolate command: %w", err)
	}

	// write everything to stdin
	if input != nil {
		eg.Go(func() error {
			_, err := io.Copy(p.Stdin(), bytes.NewReader(input))
			if err != nil {
				log.Printf("failed to write to stdin: %v", err)
			}
			return nil
		})
	}

	// read everything from stdout
	var stdout []byte
	eg.Go(func() error {
		stdout, err = io.ReadAll(p.Stdout())
		if err != nil {
			log.Printf("failed to read stdout: %v", err)
		}
		return nil
	})

	// read everything from stderr
	var stderr []byte
	eg.Go(func() error {
		stderr, err = io.ReadAll(p.Stderr())
		if err != nil {
			log.Printf("failed to read stderr: %v", err)
		}
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for isolate command: %w", err)
	}

	metrics, err := p.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for isolate command: %w", err)
	}

	return &internal.RuntimeData{
		Stdin:    input,
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: metrics.ExitCode,
		CpuMs:    metrics.CpuMillis,
		WallMs:   metrics.WallMillis,
		MemKiB:   metrics.CgMemKb,
	}, nil
}
