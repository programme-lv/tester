package isolate

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var once sync.Once

type Isolate struct {
	idsInUse []int
	mutex    sync.Mutex
}

var instance *Isolate

func GetInstance() *Isolate {
	if instance == nil {
		once.Do(func() {
			instance = &Isolate{}
		})
	}
	return instance
}

func (i *Isolate) NewBox() (*Box, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	id := 0
	for i.isIdInUse(id) {
		id++
	}

	err := i.cleanupBox(id)
	if err != nil {
		return nil, err
	}

	path, err := i.initBox(id)
	if err != nil {
		return nil, err
	}

	i.idsInUse = append(i.idsInUse, id)

	return newIsolateBox(i, id, path), nil
}

func (i *Isolate) isIdInUse(id int) bool {
	for _, usedId := range i.idsInUse {
		if usedId == id {
			return true
		}
	}
	return false
}

func (i *Isolate) cleanupBox(boxId int) error {
	cleanCmdStr := fmt.Sprintf("isolate --cg --cleanup --box-id %d", boxId)

	cleanCmd := exec.Command("/usr/bin/bash", "-c", cleanCmdStr)
	_, err := cleanCmd.CombinedOutput()
	return err
}

// initBox initializes a new box with the given id and returns the path to the box
func (i *Isolate) initBox(boxId int) (string, error) {
	initCmdStr := fmt.Sprintf("isolate --cg --init --box-id %d", boxId)

	initCmd := exec.Command("/usr/bin/bash", "-c", initCmdStr)
	cmdOutput, err := initCmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	boxPath := strings.TrimSuffix(string(cmdOutput), "\n")
	return boxPath, nil
}
