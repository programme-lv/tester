package testing

import (
	"sync"

	"github.com/programme-lv/tester/pkg/messaging"
)

func ArrangeEvalRequest(req messaging.EvaluationRequest,
	gath EvalResGatherer) (ArrangedEvaluationReq, error) {

	res := ArrangedEvaluationReq{
		Submission:  CompiledFile{},
		SubmConstrs: Constraints{},
		Checker:     CompiledFile{},
		Tests:       []TestPaths{},
		Subtasks:    []Subtask{},
	}

	// isolateInstance := isolate.GetInstance()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		// start downloading tests
	}()

	go func() {
		defer wg.Done()
		// start compiling submission
	}()

	go func() {
		defer wg.Done()
		// start compiling checker
	}()

	wg.Wait()

	return res, nil
}
