package utils

import (
	"bytes"
	"io"

	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
	"golang.org/x/sync/errgroup"
)

func RunIsolateCmd(p *isolate.Cmd, input []byte) (*internal.RuntimeData, error) {
	var eg errgroup.Group

	// write everything to stdin
	if input != nil {
		eg.Go(func() error {
			_, err := io.Copy(p.Stdin(), bytes.NewReader(input))
			if err != nil {
				return err
			}
			p.Stdin().Close()
			return nil
		})
	}

	// read everything from stdout
	var stdout []byte
	eg.Go(func() error {
		var err error
		stdout, err = io.ReadAll(p.Stdout())
		if err != nil {
			return err
		}
		return nil
	})

	// read everything from stderr
	var stderr []byte
	eg.Go(func() error {
		var err error
		stderr, err = io.ReadAll(p.Stderr())
		if err != nil {
			return err
		}
		return nil
	})

	err := eg.Wait()
	if err != nil {
		return nil, err
	}

	metrics, err := p.Wait()
	if err != nil {
		return nil, err
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
