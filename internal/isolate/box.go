package isolate

import (
	"fmt"
	"io"
	"log"
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

	var args []string
	args = append(args, "-s")
	args = append(args, "--cg")
	args = append(args, fmt.Sprintf("--box-id=%d", box.id))

	args = append(args, constraints.ToArgs()...)

	args = append(args, fmt.Sprintf("--meta=%s", process.metaFilePath))

	args = append(args, "--env=HOME=/box")
	args = append(args, "--env=PATH")

	cmdStr := fmt.Sprintf(
		"isolate %s --run -- /usr/bin/bash -c \"%s\"",
		strings.Join(args, " "),
		command,
	)

	log.Println("Running command:", cmdStr)
	cmd := exec.Command("/usr/bin/bash", "-c", cmdStr)
	cmd.Stdin = stdin

	process.cmd = cmd

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
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	err = os.Chmod(path, 0777)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, content, 0777)
	if err != nil {
		return err
	}
	return nil
}

func (box *Box) GetFile(path string) ([]byte, error) {
	path = filepath.Join(box.path, "box", path)
	return os.ReadFile(path)
}

func (box *Box) HasFile(path string) bool {
	path = filepath.Join(box.path, "box", path)
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
