package compilation

import (
	"log"

	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/internal/testing/utils"
	"github.com/programme-lv/tester/pkg/messaging"
)

func CompileSourceCode(language messaging.PLanguage, sourceCode string) (
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
	runData, err = utils.CollectProcessRuntimeData(process)
	if err != nil {
		return
	}
	log.Println("Collected compilation runtime data")

	log.Println("Compilation finished")

	if box.HasFile(language.CodeFilename) {
		log.Println("Retrieving compiled executable...")
		compiled, err = box.GetFile(language.CodeFilename)
		if err != nil {
			return
		}
		log.Println("Retrieved compiled executable")
	}

	return
}
