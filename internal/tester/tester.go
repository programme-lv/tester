package tester

import (
	"fmt"
	"os/exec"

	"github.com/programme-lv/tester/internal/checkers"
	"github.com/programme-lv/tester/internal/filestore"
)

type Tester struct {
	filestore    *filestore.FileStore
	systemInfo   string
	tlibCheckers *checkers.TestlibCompiler
}

func NewTester(filestore *filestore.FileStore, tlibCheckers *checkers.TestlibCompiler) *Tester {
	return &Tester{
		filestore:    filestore,
		systemInfo:   getSystemInfo(),
		tlibCheckers: tlibCheckers,
	}
}

// dmidecode --type memory --type processor --type cache -q
func getSystemInfo() string {
	cmd := exec.Command("dmidecode", "--type", "memory", "--type", "processor", "--type", "cache", "-q")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("failed to get system info: %v\n", err)
		panic(err)
	}
	return string(out)
}
