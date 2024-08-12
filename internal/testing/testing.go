package testing

import "fmt"

func (t *Tester) EvaluateSubmission(
	gath EvalResGatherer,
	req EvaluationRequest,
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

	gath.StartCompilation()

	return nil
}
