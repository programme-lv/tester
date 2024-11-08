package testlib

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/utils"
	pkg "github.com/programme-lv/tester/pkg"
)

type TestlibCompiler struct {
	checkerDir    string
	interactorDir string

	lock sync.Mutex
}

func NewTestlibCompiler() *TestlibCompiler {
	tc := &TestlibCompiler{
		checkerDir:    filepath.Join("var", "tester", "checkers"),
		interactorDir: filepath.Join("var", "tester", "interactors"),
	}

	err := os.MkdirAll(tc.checkerDir, 0777)
	if err != nil {
		log.Fatalf("failed to create testlib checker directory: %v", err)
	}

	err = os.MkdirAll(tc.interactorDir, 0777)
	if err != nil {
		log.Fatalf("failed to create testlib interactor directory: %v", err)
	}

	return tc
}

func (tc *TestlibCompiler) CompileChecker(sourceCode string) ([]byte, error) {
	sourceCodeSha256 := getStringSha256(sourceCode)
	tc.lock.Lock()
	defer tc.lock.Unlock()
	compiledPath := filepath.Join(tc.checkerDir, sourceCodeSha256)
	if _, err := os.Stat(compiledPath); err == nil {
		return os.ReadFile(compiledPath)
	}

	compiled, runData, err := compile(sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to compile checker: %w", err)
	}

	if runData.ExitCode != 0 {
		return nil, fmt.Errorf("checker compilation failed with exit code %d", runData.ExitCode)
	}

	err = os.WriteFile(compiledPath, compiled, 0777)
	if err != nil {
		return nil, fmt.Errorf("failed to write compiled checker: %w", err)
	}

	logPath := filepath.Join(tc.checkerDir, sourceCodeSha256+".log.json")
	runDataJson, err := json.Marshal(runData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal runtime data: %w", err)
	}
	err = os.WriteFile(logPath, runDataJson, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to write runtime data: %w", err)
	}

	cppPath := filepath.Join(tc.checkerDir, sourceCodeSha256+".cpp")
	err = os.WriteFile(cppPath, []byte(sourceCode), 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to write source code: %w", err)
	}

	return compiled, nil
}

func (tc *TestlibCompiler) CompileInteractor(sourceCode string) ([]byte, error) {
	sourceCodeSha256 := getStringSha256(sourceCode)
	tc.lock.Lock()
	defer tc.lock.Unlock()
	compiledPath := filepath.Join(tc.interactorDir, sourceCodeSha256)
	if _, err := os.Stat(compiledPath); err == nil {
		return os.ReadFile(compiledPath)
	}

	compiled, runData, err := compile(sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to compile interactor: %w", err)
	}

	if runData.ExitCode != 0 {
		return nil, fmt.Errorf("interactor compilation failed with exit code %d", runData.ExitCode)
	}

	err = os.WriteFile(compiledPath, compiled, 0777)
	if err != nil {
		return nil, fmt.Errorf("failed to write compiled interactor: %w", err)
	}

	logPath := filepath.Join(tc.interactorDir, sourceCodeSha256+".log.json")
	runDataJson, err := json.Marshal(runData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal runtime data: %w", err)
	}
	err = os.WriteFile(logPath, runDataJson, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to write runtime data: %w", err)
	}

	cppPath := filepath.Join(tc.interactorDir, sourceCodeSha256+".cpp")
	err = os.WriteFile(cppPath, []byte(sourceCode), 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to write source code: %w", err)
	}

	return compiled, nil
}

func getStringSha256(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
}

const srcCodeFname = "checker.cpp"
const compileCmd = "g++ -std=c++17 -o checker checker.cpp -I . -I /usr/include"
const compiledFname = "checker"

func compile(code string) (compiled []byte, runData *pkg.RuntimeData, err error) {
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
	err = box.AddFile(srcCodeFname, []byte(code))
	if err != nil {
		return
	}

	log.Println("Adding testlib.h to isolate box...")
	var testlibBytes []byte
	testlibPath := filepath.Join("data", "testlib.h")
	testlibBytes, err = os.ReadFile(testlibPath)
	if err != nil {
		return
	}
	err = box.AddFile("testlib.h", testlibBytes)
	if err != nil {
		return
	}

	log.Println("Running checker compilation...")
	var iCmd *isolate.Cmd
	iCmd, err = box.Command(compileCmd, nil)
	if err != nil {
		return
	}

	log.Println("Collecting compilation runtime data...")
	runData, err = utils.RunIsolateCmd(iCmd, nil)
	if err != nil {
		return
	}

	if box.HasFile(compiledFname) {
		log.Println("Retrieving compiled executable...")
		compiled, err = box.GetFile(compiledFname)
		if err != nil {
			return
		}
	}
	log.Println("Checker compilation finished!")

	return
}
