package tester

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/programme-lv/tester/internal/checkers"
	"github.com/programme-lv/tester/internal/filestore"
)

type Tester struct {
	filestore    *filestore.FileStore
	systemInfo   string
	tlibCheckers *checkers.TestlibCompiler
	logger       *log.Logger
}

func NewTester(filestore *filestore.FileStore, tlibCheckers *checkers.TestlibCompiler) *Tester {
	logger := log.New(os.Stdout, "Tester: ", log.LstdFlags|log.Lshortfile)
	return &Tester{
		filestore:    filestore,
		systemInfo:   getSystemInfo(),
		tlibCheckers: tlibCheckers,
		logger:       logger,
	}
}

// dmidecode --type memory --type processor --type cache -q
func getSystemInfo() string {
	potentialPath := filepath.Join("data", "system.txt")
	if _, err := os.Stat(potentialPath); err == nil {
		data, err := os.ReadFile(potentialPath)
		if err != nil {
			fmt.Printf("failed to read system info: %v\n", err)
			panic(err)
		}
		return string(data)
	}

	cmd := exec.Command("dmidecode", "--type", "memory", "--type", "processor", "--type", "cache", "-q")
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("failed to get system info: %v\n", err)
		panic(err)
	}
	return string(out)
}
