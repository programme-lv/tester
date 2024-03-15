package testing_test

import (
	"fmt"
	gt "testing"

	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testing/mocks"
	"github.com/programme-lv/tester/internal/testing/models"
	"github.com/programme-lv/tester/internal/testing/utils"
	"go.uber.org/mock/gomock"
)

const helloWorldCpp string = `
#include <iostream>
int main() {
	std::cout << "Hello, World!" << std::endl;
	return 0;
}
`

const testlibChecker string = `
#include "testlib.h"

using namespace std;

int main(int argc, char *argv[]) {
    setName("compare sequences of tokens");
    registerTestlibCmd(argc, argv);

    int n = 0;
    string j, p;

    while (!ans.seekEof() && !ouf.seekEof()) {
        n++;

        ans.readWordTo(j);
        ouf.readWordTo(p);

        if (j != p)
            quitf(_wa, "%d%s words differ - expected: '%s', found: '%s'", n, englishEnding(n).c_str(),
                  compress(j).c_str(), compress(p).c_str());
    }

    if (ans.seekEof() && ouf.seekEof()) {
        if (n == 1)
            quitf(_ok, "\"%s\"", compress(j).c_str());
        else
            quitf(_ok, "%d tokens", n);
    } else {
        if (ans.seekEof())
            quitf(_wa, "Participant output contains extra tokens");
        else
            quitf(_wa, "Unexpected EOF in the participants output");
    }
}
`

func TestPrepareEvalRequest_Success(t *gt.T) {
	req := getSuccessPrepareEvalRequest()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gathMock := mocks.NewMockEvalResGatherer(ctrl)

	gathMock.EXPECT().StartCompilation().Times(1)
	gathMock.EXPECT().FinishCompilation(gomock.Any()).Times(1)

	preq, err := testing.PrepareEvalRequest(&req, gathMock)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = compareTests(req.Tests, preq.Tests)
	if err != nil {
		t.Errorf("tests comparison failed: %v", err)
	}
}

func TestPrepareEvalRequest_BadSubmission(t *gt.T) {
	req := getSuccessPrepareEvalRequest()
	req.Submission = "invalid submission"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gathMock := mocks.NewMockEvalResGatherer(ctrl)

	gathMock.EXPECT().StartCompilation().Times(1)
	gathMock.EXPECT().FinishCompilation(gomock.Any()).Times(1)
	gathMock.EXPECT().FinishWithCompilationError().Times(1)

	_, err := testing.PrepareEvalRequest(&req, gathMock)
	if err == nil {
		t.Error("expected error, but got nil")
	}
}

func TestPrepareEvalRequest_BadChecker(t *gt.T) {
	req := getSuccessPrepareEvalRequest()
	req.TestlibChecker = "invalid checker"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gathMock := mocks.NewMockEvalResGatherer(ctrl)

	gathMock.EXPECT().StartCompilation().Times(1)
	gathMock.EXPECT().FinishCompilation(gomock.Any()).Times(1)
	gathMock.EXPECT().FinishWithInternalServerError(gomock.Any()).Times(1)

	_, err := testing.PrepareEvalRequest(&req, gathMock)
	if err == nil {
		t.Error("expected error, but got nil")
	}
}

func getSuccessPrepareEvalRequest() models.EvaluationRequest {
	compileCmd := "g++ -std=c++17 -O2 -o main main.cpp"
	cFname := "main"
	test0InpContent := "100 99\n"
	AnsDownlUrl := "https://proglv-dev.fra1.digitaloceanspaces.com/tests/29ef5f0b7fc0c2facd22af7e616542825331312745dfc31f37423ab0b5e005ee"
	req := models.EvaluationRequest{
		Submission: helloWorldCpp,
		PLanguage: models.PLanguage{
			ID:               "cpp17",
			FullName:         "C++17",
			CodeFilename:     "main.cpp",
			CompileCmd:       &compileCmd,
			CompiledFilename: &cFname,
			ExecCmd:          "./main",
		},
		Limits: models.Limits{
			CPUTimeMillis: 1000,
			MemKibibytes:  256 * 1024,
		},
		Tests: []models.TestRef{
			{
				ID:          1,
				InContent:   &test0InpContent,
				InSHA256:    "9c1f9534ed3f91538f3668f886aa4bb6d158dfdca7790b93fc2bd4f2c9ede944",
				InDownlUrl:  nil,
				AnsContent:  nil,
				AnsSHA256:   "29ef5f0b7fc0c2facd22af7e616542825331312745dfc31f37423ab0b5e005ee",
				AnsDownlUrl: &AnsDownlUrl,
			},
		},
		TestlibChecker: testlibChecker,
	}
	return req
}

func compareTests(original []models.TestRef, prepared []models.Test) error {
	if len(original) != len(prepared) {
		return fmt.Errorf("expected %d tests, but got %d", len(original), len(prepared))
	}

	for i := 0; i < len(original); i++ {
		otest := original[i] // original test
		ptest := prepared[i] // prepared test
		if otest.ID != ptest.ID {
			return fmt.Errorf("test %d: expected ID %d, but got %d", i, otest.ID, ptest.ID)
		}

		err := utils.VerifySha256(ptest.InputSHA, otest.InSHA256)
		if err != nil {
			return fmt.Errorf("test %d input: %v", i, err)
		}
		if otest.InContent != nil {
			err = utils.VerifyContent(ptest.InputSHA, []byte(*otest.InContent))
			if err != nil {
				return fmt.Errorf("test %d input: %v", i, err)
			}
		}
		err = utils.VerifySha256(ptest.AnswerSHA, otest.AnsSHA256)
		if err != nil {
			return fmt.Errorf("test %d answer: %v", i, err)
		}
		if otest.AnsContent != nil {
			err = utils.VerifyContent(ptest.AnswerSHA, []byte(*otest.AnsContent))
			if err != nil {
				return fmt.Errorf("test %d answer: %v", i, err)
			}
		}
	}

	return nil
}
