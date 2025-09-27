package tester

import (
	"log"
	"log/slog"
	"os"

	"github.com/programme-lv/tester/internal/filecache"
	"github.com/programme-lv/tester/internal/testlib"
)

type Tester struct {
	filestore    *filecache.FileStore
	systemInfo   string
	tlibCheckers *testlib.TestlibCompiler
	testlibHStr  string
	loggerOld    *log.Logger
	logger       *slog.Logger
}

func NewTester(
	filestore *filecache.FileStore,
	tlibCheckers *testlib.TestlibCompiler,
	systemInfoTxt string,
	testlibHStr string) *Tester {
	logger := log.New(os.Stdout, "Tester: ", log.LstdFlags|log.Lshortfile)
	return &Tester{
		filestore:    filestore,
		systemInfo:   systemInfoTxt,
		tlibCheckers: tlibCheckers,
		testlibHStr:  testlibHStr,
		loggerOld:    logger,
		logger:       slog.Default(),
	}
}
