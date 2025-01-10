package isolate

import (
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
)

type Cmd struct {
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	started      bool
	metaFilePath string
	Constraints  Constraints
}

// func (process *Cmd) CombinedOutput() (*Metrics, []byte, error) {
// 	combinedOut, err := process.cmd.CombinedOutput()
// 	if err != nil {
// 		var exitErr *exec.ExitError
// 		if !errors.As(err, &exitErr) {
// 			return nil, nil, err
// 		}
// 	}

// 	metaFileBytes, err := os.ReadFile(process.metaFilePath)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	metrics, err := parseMetaFile(metaFileBytes)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	return metrics, combinedOut, nil
// }

func (process *Cmd) Start() error {
	if process.started {
		panic("process should not be started twice")
	}
	process.started = true

	var err error
	process.stdin, err = process.cmd.StdinPipe()
	if err != nil {
		return err
	}

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

func (process *Cmd) Wait() (*Metrics, error) {
	if !process.started {
		panic("process should be started before waiting")
	}

	log.Printf("Waiting for isolate command to finish")
	err := process.cmd.Wait()

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return nil, err
		}
	}
	log.Printf("Isolate command finished")

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

func (process *Cmd) Stdin() io.WriteCloser {
	if process.stdin == nil {
		panic("process should be started before retrieving stdin")
	}
	return process.stdin
}

func (process *Cmd) Stdout() io.ReadCloser {
	if process.stdout == nil {
		panic("process should be started before retrieving stdout")
	}
	return process.stdout
}

func (process *Cmd) Stderr() io.ReadCloser {
	if process.stderr == nil {
		panic("process should be started before retrieving stderr")
	}
	return process.stderr
}
