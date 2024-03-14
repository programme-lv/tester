package compilation

import (
	"log"

	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/storage"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/internal/testing/utils"
)

const testlibCheckerCodeFilename = "checker.cpp"
const testlibCheckerCompileCmd = "g++ -std=c++17 -o checker checker.cpp -I . -I /usr/include"
const testlibCheckerCompiledFilename = "checker"

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
	err = box.AddFile(testlibCheckerCodeFilename, []byte(code))
	if err != nil {
		return
	}

	log.Println("Adding testlib.h to isolate box...")
	var testlibBytes []byte
	s, err := storage.GetInstance()
	if err != nil {
		return
	}
	testlibBytes, err = s.GetTestlib()
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
