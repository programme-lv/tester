package testing

import (
	"fmt"

	"github.com/programme-lv/tester/internal"
)

func (t *Tester) EvaluateSubmission(
	gath EvalResGatherer,
	req internal.EvaluationRequest,
) error {
	gath.StartEvaluation(t.systemInfo)
	for _, test := range req.Tests {
		err := t.filestore.ScheduleDownloadFromS3(test.InputSha256, *test.InputS3Uri)
		if err != nil {
			return fmt.Errorf("failed to schedule file for download: %w", err)
		}

		err = t.filestore.ScheduleDownloadFromS3(test.AnswerSha256, *test.AnswerS3Uri)
		if err != nil {
			return fmt.Errorf("failed to schedule file for download: %w", err)
		}
	}

	// compile testlib checker
	// req.TestlibChecker

	if req.Language.CompileCommand != nil {
		gath.StartCompilation()

	}

	return nil
}

// func compileSubmission(req *internal.EvaluationRequest) (
// 	*models.ExecutableFile, *models.RuntimeData, error) {

// 	code := req.Submission
// 	pLang := req.PLanguage

// 	if pLang.CompileCmd == nil {
// 		return &models.ExecutableFile{
// 			Content:  []byte(code),
// 			Filename: pLang.CodeFilename,
// 			ExecCmd:  pLang.ExecCmd,
// 		}, nil, nil
// 	}

// 	fname := pLang.CodeFilename
// 	cCmd := *pLang.CompileCmd
// 	cFname := *pLang.CompiledFilename

// 	compiled, runData, err := compilation.CompileSourceCode(
// 		code, fname, cCmd, cFname)

// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	return &models.ExecutableFile{
// 		Content:  compiled,
// 		Filename: *pLang.CompiledFilename,
// 		ExecCmd:  pLang.ExecCmd,
// 	}, runData, nil
// }
