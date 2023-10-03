package isolate

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
)

type Process struct {
	cmd          *exec.Cmd
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	metaFilePath string
}

func (process *Process) Wait() (*Metrics, error) {
	err := process.cmd.Wait()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if _, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				// fmt.Printf("Exit Status: %d\n", status.ExitStatus())
			} else {
				// fmt.Println("Cmd failed but was unable to determine exit status.")
			}
		} else {
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
