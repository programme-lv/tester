package tester

import (
	"fmt"
	"log"

	"github.com/programme-lv/tester/internal"
	"github.com/programme-lv/tester/internal/isolate"
	"github.com/programme-lv/tester/internal/utils"
)

func (t *Tester) EvaluateSubmission(
	gath EvalResGatherer,
	req internal.EvaluationRequest,
) error {
	gath.StartEvaluation(t.systemInfo)
	for _, test := range req.Tests {
		err := t.filestore.ScheduleDownloadFromS3(test.InputSha256, *test.InputS3Uri)
		if err != nil {
			errMsg := fmt.Errorf("failed to schedule file for download: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}

		err = t.filestore.ScheduleDownloadFromS3(test.AnswerSha256, *test.AnswerS3Uri)
		if err != nil {
			errMsg := fmt.Errorf("failed to schedule file for download: %w", err)
			gath.FinishEvaluation(errMsg)
			return errMsg
		}
	}

	tlibChecker, err := t.tlibCheckers.GetExecutable(req.TestlibChecker)
	if err != nil {
		errMsg := fmt.Errorf("failed to get testlib checker: %w", err)
		gath.FinishEvaluation(errMsg)
		return errMsg
	}

	var compiled []byte
	if req.Language.CompileCommand != nil {
		gath.StartCompilation()
		var runData *internal.RuntimeData
		compiled, runData, err = compileSourceCode(
			req.Submission,
			req.Language.SourceCodeFname,
			*req.Language.CompileCommand,
			*req.Language.CompiledFilename,
		)
		if err != nil {
			errMsg := fmt.Errorf("failed to compile source code: %w", err)
			gath.FinishEvaluation(errMsg)
		}
		gath.FinishCompilation(runData)
	}

	gath.StartTesting()
	// TODO: testing
	fmt.Printf("Compiled: %v\n", len(compiled))
	fmt.Printf("Testlib checker: %v\n", len(tlibChecker))
	gath.FinishTesting()

	gath.FinishEvaluation(nil)

	return nil
}

func compileSourceCode(code, fname, compileCmd, cFname string) (
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

	if box.HasFile(cFname) {
		log.Println("Retrieving compiled executable...")
		compiled, err = box.GetFile(cFname)
		if err != nil {
			return
		}
	}

	log.Println("Compilation finished!")
	return
}
