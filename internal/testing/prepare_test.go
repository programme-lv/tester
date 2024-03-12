package testing_test

import (
	gt "testing"

	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testing/mocks"
	"github.com/programme-lv/tester/pkg/messaging"
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
	compileCmd := "g++ -std=c++17 -O2 -o main main.cpp"
	req := messaging.EvaluationRequest{
		Submission: helloWorldCpp,
		PLanguage: messaging.PLanguage{
			ID:           "cpp17",
			FullName:     "C++17",
			CodeFilename: "main.cpp",
			CompileCmd:   &compileCmd,
			ExecCmd:      "./main",
		},
		Limits: messaging.Limits{
			CPUTimeMillis: 1000,
			MemKibibytes:  256 * 1024,
		},
		TestlibChecker: testlibChecker,
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gathMock := mocks.NewMockEvalResGatherer(ctrl)

	// type PreparedEvaluationReq struct {
	// 	Submission  ExecutableFile
	// 	SubmConstrs Constraints
	// 	Checker     ExecutableFile

	// 	Tests    []TestPaths
	// 	Subtasks []Subtask
	// }
	gathMock.EXPECT().StartCompilation().Times(1)
	gathMock.EXPECT().FinishCompilation(gomock.Any()).Times(1)

	_, err := testing.PrepareEvalRequest(req, gathMock)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// try to run the compiled submission & code
	// verify that the tests correspond to their SHA

}
