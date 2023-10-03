package isolate

import (
	"io"
	"log/slog"
	"os"
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
	constraints *RuntimeConstraints) (*Process, error) {
	if constraints == nil {
		c := DefaultRuntimeConstraints()
		constraints = &c
	}
	box.logger.Info("running command in box", slog.String("command", command),
		slog.String("constraints", strings.Join(constraints.ToArgs(), " ")))

	return box.isolate.StartCommand(box.id, command, stdin, *constraints)
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
