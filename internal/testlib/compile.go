package testlib

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/utils"
	"github.com/programme-lv/tester/internal/xdg"
)

type TestlibCompiler struct {
	checkerDir    string
	interactorDir string

	lock sync.Mutex
}

func NewTestlibCompiler() *TestlibCompiler {
	// Initialize XDG directories
	xdgDirs := xdg.NewXDGDirs()

	// Use XDG cache directory for compiled checkers and interactors
	// These are cached compiled binaries that can be regenerated
	tc := &TestlibCompiler{
		checkerDir:    xdgDirs.AppCacheDir("tester/checkers"),
		interactorDir: xdgDirs.AppCacheDir("tester/interactors"),
	}

	err := xdgDirs.EnsureDir(tc.checkerDir)
	if err != nil {
		panic(fmt.Sprintf("failed to create testlib checker directory: %v", err))
	}

	err = xdgDirs.EnsureDir(tc.interactorDir)
	if err != nil {
		panic(fmt.Sprintf("failed to create testlib interactor directory: %v", err))
	}

	return tc
}

func (tc *TestlibCompiler) CompileChecker(sourceCode string, testlibHeaderStr string) ([]byte, error) {
	sourceCodeSha256 := getStringSha256(sourceCode)
	tc.lock.Lock()
	defer tc.lock.Unlock()
	compiledPath := filepath.Join(tc.checkerDir, sourceCodeSha256)
	if _, err := os.Stat(compiledPath); err == nil {
		return os.ReadFile(compiledPath)
	}

	compiled, runData, err := compile(sourceCode, testlibHeaderStr)
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

func (tc *TestlibCompiler) CompileInteractor(sourceCode string, testlibHeaderStr string) ([]byte, error) {
	sourceCodeSha256 := getStringSha256(sourceCode)
	tc.lock.Lock()
	defer tc.lock.Unlock()
	compiledPath := filepath.Join(tc.interactorDir, sourceCodeSha256)
	if _, err := os.Stat(compiledPath); err == nil {
		return os.ReadFile(compiledPath)
	}

	compiled, runData, err := compile(sourceCode, testlibHeaderStr)
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

func compile(code string, testlibHeaderStr string) (compiled []byte, runData *internal.RunData, err error) {
	isolateInstance := isolate.GetInstance()
	var box *isolate.Box
	box, err = isolateInstance.NewBox()
	if err != nil {
		err = fmt.Errorf("failed to create isolate box: %w", err)
		return
	}

	defer func(box *isolate.Box) {
		_ = box.Close()
	}(box)

	err = box.AddFile(srcCodeFname, []byte(code))
	if err != nil {
		err = fmt.Errorf("failed to add checker code to isolate box: %w", err)
		return
	}

	err = box.AddFile("testlib.h", []byte(testlibHeaderStr))
	if err != nil {
		err = fmt.Errorf("failed to add testlib.h to isolate box: %w", err)
		return
	}

	var iCmd *isolate.Cmd
	iCmd, err = box.Command(compileCmd, nil)
	if err != nil {
		err = fmt.Errorf("failed to create isolate command: %w ", err)
		return
	}

	runData, err = utils.RunIsolateCmd(iCmd, nil)
	if err != nil {
		err = fmt.Errorf("failed to collect runtime data: %s, %w ", iCmd.String(), err)
		return
	}

	if box.HasFile(compiledFname) {
		compiled, err = box.GetFile(compiledFname)
		if err != nil {
			err = fmt.Errorf("failed to get compiled executable: %w", err)
			return
		}
	}

	return
}
