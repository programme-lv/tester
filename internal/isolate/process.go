package isolate

import (
	"errors"
	"io"
	"os"
	"os/exec"
)

type Process struct {
	cmd          *exec.Cmd
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	metaFilePath string
}

func (process *Process) CombinedOutput() (*Metrics, []byte, error) {
	combinedOut, err := process.cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, nil, err
		}
	}

	metaFileBytes, err := os.ReadFile(process.metaFilePath)
	if err != nil {
		return nil, nil, err
	}

	metrics, err := parseMetaFile(metaFileBytes)
	if err != nil {
		return nil, nil, err
	}

	return metrics, combinedOut, nil
}

func (process *Process) Start() error {
	var err error
	process.stdout, err = process.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	process.stderr, err = process.cmd.StderrPipe()
	if err != nil {
		return err
	}

	return process.cmd.Start()
}

func (process *Process) Wait() (*Metrics, error) {
	err := process.cmd.Wait()

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, err
		}
	}

	metaFileBytes, err := os.ReadFile(process.metaFilePath)
	if err != nil {
		return nil, err
	}

	metrics, err := parseMetaFile(metaFileBytes)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func (process *Process) Stdout() io.ReadCloser {
	return process.stdout
}

func (process *Process) Stderr() io.ReadCloser {
	return process.stderr
}
