package utils

import (
	"bytes"
	"fmt"
	"io"

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
			_, _ = io.Copy(p.Stdin(), bytes.NewReader(input))
			// if err != nil {
			// 	// return err if it's not broken pipe
			// 	if err != io.ErrClosedPipe {
			// 		return fmt.Errorf("failed to write to stdin: %w", err)
			// 	}
			// }
			return nil
		})
	}

	// read everything from stdout
	var stdout []byte
	eg.Go(func() error {
		var err error
		stdout, err = io.ReadAll(p.Stdout())
		if err != nil {
			if err != io.ErrClosedPipe {
				return fmt.Errorf("failed to read stdout: %w", err)
			}
		}
		return nil
	})

	// read everything from stderr
	var stderr []byte
	eg.Go(func() error {
		var err error
		stderr, err = io.ReadAll(p.Stderr())
		if err != nil {
			if err != io.ErrClosedPipe {
				return fmt.Errorf("failed to read stderr: %w", err)
			}
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
		Stdout:        stdout,
		Stderr:        stderr,
		ExitCode:      metrics.ExitCode,
		CpuMillis:     metrics.CpuMillis,
		WallMillis:    metrics.WallMillis,
		MemoryKiBytes: metrics.CgMemKb,
	}, nil
}
