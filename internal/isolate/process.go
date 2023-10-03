package isolate

import (
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

func (process *Process) Wait() (*Metrics, error) {
	if err := process.cmd.Wait(); err != nil {
		return nil, err
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
