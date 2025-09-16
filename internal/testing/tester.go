package testing

import (
	"log"
	"os"

	"github.com/programme-lv/tester/internal/filestore"
	"github.com/programme-lv/tester/internal/testlib"
)

type Tester struct {
	filestore    *filestore.FileStore
	systemInfo   string
	tlibCheckers *testlib.TestlibCompiler
	testlibHStr  string
	logger       *log.Logger
}

func NewTester(
	filestore *filestore.FileStore,
	tlibCheckers *testlib.TestlibCompiler,
	systemInfoTxt string,
	testlibHStr string) *Tester {
	logger := log.New(os.Stdout, "Tester: ", log.LstdFlags|log.Lshortfile)
	return &Tester{
		filestore:    filestore,
		systemInfo:   systemInfoTxt,
		tlibCheckers: tlibCheckers,
		testlibHStr:  testlibHStr,
		logger:       logger,
	}
}
