package utils

import (
	"bytes"
	"fmt"
	"io"

	"github.com/programme-lv/tester/api"
	"github.com/programme-lv/tester/internal/isolate"
	"golang.org/x/sync/errgroup"
)

func RunIsolateCmd(p *isolate.Cmd, input []byte) (*api.RuntimeData, error) {
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
			}
			_ = p.Stdin().Close()
			return nil
		})
	}

	// read everything from stdout
	var stdout []byte
	eg.Go(func() error {
		stdout, err = io.ReadAll(p.Stdout())
		if err != nil {
		}
		return nil
	})

	// read everything from stderr
	var stderr []byte
	eg.Go(func() error {
		stderr, err = io.ReadAll(p.Stderr())
		if err != nil {
		}
		return nil
	})

	metrics, err := p.Wait()
	if err != nil {
		return nil, fmt.Errorf("wait for isolate command: %w", err)
	}

	err = eg.Wait()
	if err != nil {
		return nil, fmt.Errorf("wait for isolate command: %w", err)
	}

	return &api.RuntimeData{
		Stdin:         string(input),
		Stdout:        string(stdout),
		Stderr:        string(stderr),
		ExitCode:      metrics.ExitCode,
		CpuMillis:     metrics.CpuMillis,
		WallMillis:    metrics.WallMillis,
		RamKiBytes:    metrics.CgMemKb,
		IsolateStatus: metrics.Status,
		CtxSwV:        metrics.CswVoluntary,
		CtxSwF:        metrics.CswForced,
		ExitSignal:    metrics.ExitSig,
		IsolateMsg:    metrics.Message,
		CgOomKilled:   metrics.CgOomKilled,
	}, nil
}
