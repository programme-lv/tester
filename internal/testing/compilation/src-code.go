package compilation

import (
	"log"

	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/internal/testing/utils"
)

func CompileSourceCode(code, fname, compileCmd, cFname string) (
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
		err = box.Close()
		if err != nil {
			log.Printf("Failed to close isolate box: %v", err)
		}
	}(box)

	log.Println("Adding source code to isolate box...")
	err = box.AddFile(fname, []byte(code))
	if err != nil {
		return
	}

	log.Println("Running compilation...")
	var process *isolate.Process
	process, err = box.Run(compileCmd, nil, nil)
	if err != nil {
		return
	}

	log.Println("Collecting compilation runtime data...")
	runData, err = utils.CollectProcessRuntimeData(process)
	if err != nil {
		return
	}

	log.Println("Retrieving compiled executable...")
	compiled, err = box.GetFile(cFname)
	if err != nil {
		return
	}

	log.Println("Compilation finished!")
	return
}
