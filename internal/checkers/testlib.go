package checkers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/utils"
)

type TestlibCompiler struct {
	tlibCheckerDir string
	checkerSyncMap sync.Map
}

func NewTestlibCheckerCompiler() *TestlibCompiler {
	tc := &TestlibCompiler{
		tlibCheckerDir: filepath.Join("var", "tester", "checkers", "testlib"),
	}

	err := os.MkdirAll(tc.tlibCheckerDir, 0777)
	if err != nil {
		log.Fatalf("failed to create testlib checker directory: %v", err)
	}

	return tc
}

func (cs *TestlibCompiler) GetExecutable(sourceCode string) ([]byte, error) {
	sourceCodeSha256 := getStringSha256(sourceCode)
	c := make(chan struct{}, 1)
	_, exists := cs.checkerSyncMap.LoadOrStore(sourceCode, c)
	if exists {
		close(c)
		return os.ReadFile(filepath.Join(cs.tlibCheckerDir, sourceCodeSha256))
	} else {
		if _, err := os.Stat(filepath.Join(cs.tlibCheckerDir, sourceCodeSha256)); err == nil {
			close(c)
			fmt.Printf("Checker %s already exists\n", sourceCodeSha256)
			return os.ReadFile(filepath.Join(cs.tlibCheckerDir, sourceCodeSha256))
		} else {
			fmt.Printf("Checker %s does not exist\n", sourceCodeSha256)
		}

		compiled, runData, err := compileTestlibChecker(sourceCode)
		if err != nil {
			return nil, fmt.Errorf("failed to compile checker: %w", err)
		}

		if runData.ExitCode != 0 {
			return nil, fmt.Errorf("checker compilation failed with exit code %d", runData.ExitCode)
		}

		err = os.WriteFile(filepath.Join(cs.tlibCheckerDir, sourceCodeSha256), compiled, 0777)
		if err != nil {
			return nil, fmt.Errorf("failed to write compiled checker: %w", err)
		}

		runDataJson, err := json.Marshal(runData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal runtime data: %w", err)
		}

		err = os.WriteFile(filepath.Join(cs.tlibCheckerDir, sourceCodeSha256+".log"), runDataJson, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to write runtime data: %w", err)
		}

		err = os.WriteFile(filepath.Join(cs.tlibCheckerDir, sourceCodeSha256+".cpp"), []byte(sourceCode), 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to write source code: %w", err)
		}

		close(c)

		return compiled, nil
	}
}

func getStringSha256(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
}

const testlibCheckerCodeFilename = "checker.cpp"
const testlibCheckerCompileCmd = "g++ -std=c++17 -o checker checker.cpp -I . -I /usr/include"
const testlibCheckerCompiledFilename = "checker"

func compileTestlibChecker(code string) (
	compiled []byte,
	runData *internal.RuntimeData,
	err error,
) {
	isolateInstance := isolate.GetInstance()

	log.Println("Creating isolate box...")
	var box *isolate.Box
	box, err = isolateInstance.NewBox()
	if err != nil {
		return
	}
	log.Println("Created isolate box:", box.Path())

	defer func(box *isolate.Box) {
		_ = box.Close()
	}(box)

	log.Println("Adding checker code to isolate box...")
	err = box.AddFile(testlibCheckerCodeFilename, []byte(code))
	if err != nil {
		return
	}

	log.Println("Adding testlib.h to isolate box...")
	var testlibBytes []byte
	testlibBytes, err = readTestlibHeader()
	if err != nil {
		return
	}
	err = box.AddFile("testlib.h", testlibBytes)
	if err != nil {
		return
	}

	log.Println("Running checker compilation...")
	var process *isolate.Process
	process, err = box.Run(testlibCheckerCompileCmd, nil, nil)
	if err != nil {
		return
	}

	log.Println("Collecting compilation runtime data...")
	runData, err = utils.CollectProcessRuntimeData(process)
	if err != nil {
		return
	}

	if box.HasFile(testlibCheckerCompiledFilename) {
		log.Println("Retrieving compiled executable...")
		compiled, err = box.GetFile(testlibCheckerCompiledFilename)
		if err != nil {
			return
		}
	}
	log.Println("Checker compilation finished!")

	return
}

func readTestlibHeader() ([]byte, error) {
	path := filepath.Join("data", "testlib.h")
	return os.ReadFile(path)
}
