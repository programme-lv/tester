package isolate

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Box struct {
	id      int
	path    string
	isolate *Isolate
}

func newIsolateBox(isolate *Isolate, id int, path string) *Box {
	return &Box{
		id:      id,
		path:    path,
		isolate: isolate,
	}
}

func (box *Box) Id() int {
	return box.id
}

func (box *Box) Path() string {
	return box.path
}

func (box *Box) Close() error {
	return box.isolate.eraseBox(box.id)
}

func (box *Box) Run(
	command string,
	stdin io.ReadCloser,
	constraints *Constraints) (*Process, error) {

	if constraints == nil {
		c := DefaultConstraints()
		constraints = &c
	}

	var process *Process = &Process{}

	err := assignMetaFilePathToProcess(process)
	if err != nil {
		return nil, err
	}

	args := []string{"--env=HOME=/box", "--meta=" + process.metaFilePath}
	args = append(args, constraints.ToArgs()...)

	cmdStr := fmt.Sprintf(
		"isolate --cg --box-id %d %s --run /usr/bin/env %s",
		box.id,
		strings.Join(args, " "),
		command,
	)

	cmd := exec.Command("/usr/bin/bash", "-c", cmdStr)
	cmd.Stdin = stdin
	process.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return process, err
	}
	process.stderr, err = cmd.StderrPipe()
	if err != nil {
		return process, err
	}
	process.cmd = cmd

	if err = cmd.Start(); err != nil {
		return process, err
	}

	return process, err
}

func assignMetaFilePathToProcess(process *Process) error {
	tempFilePath, err := newTempIsolateFilePath()
	if err != nil {
		return err
	}
	process.metaFilePath = tempFilePath
	return nil
}

func newTempIsolateFilePath() (string, error) {
	file, err := os.CreateTemp("", "isolate.*.txt")
	if err != nil {
		return "", err
	}
	err = file.Close()
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

func (box *Box) AddFile(path string, content []byte) error {
	path = filepath.Join(box.path, "box", path)
	_, err := os.Create(path)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, content, 0644)
	if err != nil {
		return err
	}
	return nil
}
