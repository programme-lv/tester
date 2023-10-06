package testing

import (
	"github.com/programme-lv/tester/internal/database"
	"github.com/programme-lv/tester/internal/isolate"
	"log"
	"os"
)

const testLibUrl = "https://raw.githubusercontent.com/MikeMirzayanov/testlib/master/testlib.h"

const testlibCodeFilename = "main.cpp"
const testlibCompileCmd = "g++ -std=c++17 -o main main.cpp -I . -I /usr/include"
const testlibCompiledFilename = "main"

func compileSourceCode(language *database.ProgrammingLanguage, sourceCode string) (
	compiled []byte,
	runData *RuntimeData,
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

	log.Println("Adding source code to isolate box...")
	err = box.AddFile(language.CodeFilename, []byte(sourceCode))
	if err != nil {
		return
	}
	log.Println("Added source code to isolate box")

	log.Println("Running compilation...")
	var process *isolate.Process
	process, err = box.Run(*language.CompileCmd, nil, nil)
	if err != nil {
		return
	}
	log.Println("Ran compilation command")

	log.Println("Collecting compilation runtime data...")
	runData, err = collectProcessRuntimeData(process)
	if err != nil {
		return
	}
	log.Println("Collected compilation runtime data")

	log.Println("Compilation finished")

	if box.HasFile(*language.CompiledFilename) {
		log.Println("Retrieving compiled executable...")
		compiled, err = box.GetFile(*language.CompiledFilename)
		if err != nil {
			return
		}
		log.Println("Retrieved compiled executable")
	}

	return
}

func compileTestlibChecker(code string) (
	compiled []byte,
	runData *RuntimeData,
	err error,
) {

	log.Println("Ensuring testLib exists...")
	err = ensureTestlibExistsInCache()
	if err != nil {
		return
	}
	log.Println("TestLib exists / was downloaded")

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
	err = box.AddFile(testlibCodeFilename, []byte(code))
	if err != nil {
		return
	}
	log.Println("Added checker code to isolate box")

	log.Println("Adding testLib to isolate box...")
	var testlibBytes []byte
	testlibBytes, err = os.ReadFile(testLibCachePath)
	if err != nil {
		return
	}
	err = box.AddFile("testlib.h", testlibBytes)
	if err != nil {
		return
	}

	log.Println("Running checker compilation...")
	var process *isolate.Process
	process, err = box.Run(testlibCompileCmd, nil, nil)
	if err != nil {
		return
	}
	log.Println("Ran checker compilation command")

	log.Println("Collecting compilation runtime data...")
	runData, err = collectProcessRuntimeData(process)
	if err != nil {
		return
	}
	log.Println("Collected compilation runtime data")

	log.Println("Compilation finished")

	if box.HasFile(testlibCompiledFilename) {
		log.Println("Retrieving compiled executable...")
		compiled, err = box.GetFile(testlibCompiledFilename)
		if err != nil {
			return
		}
		log.Println("Retrieved compiled executable")
	}
	return
}
