package isolate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/programme-lv/tester/internal/xdg"
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

func (box *Box) Command(
	command string, constraints *Constraints) (*Cmd, error) {

	var isolateCmd *Cmd = &Cmd{}
	if constraints != nil {
		isolateCmd.Constraints = *constraints
	} else {
		isolateCmd.Constraints = DefaultConstraints()
	}

	tempFilePath, err := newTempIsolateFilePath()
	if err != nil {
		return nil, err
	}
	isolateCmd.metaFilePath = tempFilePath

	var args []string
	args = append(args, "-s")
	args = append(args, "--cg")
	args = append(args, fmt.Sprintf("--box-id=%d", box.id))

	args = append(args, isolateCmd.Constraints.ToArgs()...)

	args = append(args, fmt.Sprintf("--meta=%s", isolateCmd.metaFilePath))

	args = append(args, "--env=HOME=/box")
	args = append(args, "--env=PATH")

	if _, err := os.Stat("/etc/alternatives"); err == nil {
		// for some reason, java on ubuntu symlinks to /etc/alternatives/java
		args = append(args, "--dir=/etc/alternatives")
	}

	cmdStr := fmt.Sprintf(
		"isolate %s --run -- /usr/bin/bash -c \"%s\"",
		strings.Join(args, " "),
		command,
	)

	goCmd := exec.Command("/usr/bin/bash", "-c", cmdStr)

	isolateCmd.cmd = goCmd
	return isolateCmd, err
}

func newTempIsolateFilePath() (string, error) {
	// Use XDG runtime directory for isolate temporary files
	xdgDirs := xdg.NewXDGDirs()
	tempDir := xdgDirs.AppRuntimeDir("tester/isolate")
	err := xdgDirs.EnsureRuntimeDir(tempDir)
	if err != nil {
		// Fallback to system temp if XDG runtime dir fails
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

	file, err := os.CreateTemp(tempDir, "isolate.*.txt")
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
