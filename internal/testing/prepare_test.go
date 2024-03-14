package testing_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	gt "testing"

	"github.com/programme-lv/tester/internal/storage"
	"github.com/programme-lv/tester/internal/testing"
	"github.com/programme-lv/tester/internal/testing/mocks"
	"github.com/programme-lv/tester/pkg/messaging"
	"github.com/stretchr/testify/assert"
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
	cFname := "main"
	test0InpContent := "100 99\n"
	AnsDownlUrl := "https://proglv-dev.fra1.digitaloceanspaces.com/tests/29ef5f0b7fc0c2facd22af7e616542825331312745dfc31f37423ab0b5e005ee"
	req := messaging.EvaluationRequest{
		Submission: helloWorldCpp,
		PLanguage: messaging.PLanguage{
			ID:               "cpp17",
			FullName:         "C++17",
			CodeFilename:     "main.cpp",
			CompileCmd:       &compileCmd,
			CompiledFilename: &cFname,
			ExecCmd:          "./main",
		},
		Limits: messaging.Limits{
			CPUTimeMillis: 1000,
			MemKibibytes:  256 * 1024,
		},
		Tests: []messaging.TestRef{
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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	gathMock := mocks.NewMockEvalResGatherer(ctrl)

	gathMock.EXPECT().StartCompilation().Times(1)
	gathMock.EXPECT().FinishCompilation(gomock.Any()).Times(1)

	pReq, err := testing.PrepareEvalRequest(req, gathMock)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	assert.Equal(t, len(req.Tests), len(pReq.Tests))
	for i := 0; i < len(req.Tests); i++ {
		otest := req.Tests[i]  // original test
		ptest := pReq.Tests[i] // prepared test
		assert.Equal(t, otest.ID, ptest.ID)

		err = verifySha256(ptest.InputSHA, otest.InSHA256)
		if err != nil {
			t.Errorf("test %d input: %v", i, err)
			return
		}
		if otest.InContent != nil {
			err = verifyContent(ptest.InputSHA, []byte(*otest.InContent))
			if err != nil {
				t.Errorf("test %d input: %v", i, err)
			}
		}
		err = verifySha256(ptest.AnswerSHA, otest.AnsSHA256)
		if err != nil {
			t.Errorf("test %d answer: %v", i, err)
			return
		}
		if otest.AnsContent != nil {
			err = verifyContent(ptest.AnswerSHA, []byte(*otest.AnsContent))
			if err != nil {
				t.Errorf("test %d answer: %v", i, err)
			}
		}
	}
}

func verifyContent(fname string, expected []byte) error {
	s, err := storage.GetInstance()
	if err != nil {
		return err
	}

	content, err := s.GetTextFile(fname)
	if err != nil {
		return err
	}
	if string(content) != string(expected) {
		return fmt.Errorf("file %s has content %s, but expected %s", fname, content, expected)
	}

	return nil
}

func verifySha256(fname string, expected string) error {
	s, err := storage.GetInstance()
	if err != nil {
		return err
	}

	file, err := s.GetTextFile(fname)
	if err != nil {
		return err
	}

	h := sha256.New()
	if _, err := io.Copy(h, bytes.NewReader(file)); err != nil {
		return err
	}

	sha256sum := fmt.Sprintf("%x", h.Sum(nil))
	if sha256sum != expected {
		return fmt.Errorf("file %s has sha256 %s, but expected %s", fname, sha256sum, expected)
	}

	return nil
}
