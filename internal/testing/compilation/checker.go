package compilation

import (
	"log"

	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/storage"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/internal/testing/utils"
)

const testlibCodeFilename = "main.cpp"
const testlibCompileCmd = "g++ -std=c++17 -o main main.cpp -I . -I /usr/include"
const testlibCompiledFilename = "main"

func CompileTestlibChecker(code string) (
	compiled []byte,
	runData *models.RuntimeData,
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
	err = box.AddFile(testlibCodeFilename, []byte(code))
	if err != nil {
		return
	}
	log.Println("Added checker code to isolate box")

	log.Println("Adding testLib to isolate box...")
	var testlibBytes []byte
	testlibBytes, err = storage.GetTestlib()
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
	runData, err = utils.CollectProcessRuntimeData(process)
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
