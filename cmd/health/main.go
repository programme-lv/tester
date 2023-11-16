package main

import (
	jet "github.com/go-jet/jet/v2/postgres"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/programme-lv/tester/internal/database/proglv/public/model"
	"github.com/programme-lv/tester/internal/database/proglv/public/table"
	"github.com/programme-lv/tester/internal/environment"
	"github.com/programme-lv/tester/internal/isolate"
	"io"
	"log"
	"os/exec"
	"strings"
)

type feedbackRow struct {
	unit    string
	health  int
	message string
}

func main() {
	ensureIsolateOk()

	ensureLanguagesOk()

}

func ensureIsolateOk() {
	isolateCmd := exec.Command("isolate", "--cg", "--cleanup")
	log.Printf("Running %v...", isolateCmd.Args)
	err := isolateCmd.Run()
	panicOnError(err)
	log.Printf("Finished %v OK", isolateCmd.Args)
}

func ensureLanguagesOk() {
	languages := fetchLanguages()

	isolateInstance := isolate.GetInstance()

	for _, language := range languages {
		box, err := isolateInstance.NewBox()
		panicOnError(err)

		err = box.AddFile(language.CodeFilename, []byte(*language.HelloWorldCode))
		panicOnError(err)

		var process *isolate.Process
		process, err = box.Run(language.ExecuteCmd, io.NopCloser(strings.NewReader("")), nil)

		stdout := process.Stdout()
		stderr := process.Stderr()

		var stdoutBytes []byte
		stdoutBytes, err = io.ReadAll(stdout)
		panicOnError(err)
		stdoutStr := string(stdoutBytes)

		var stderrBytes []byte
		stderrBytes, err = io.ReadAll(stderr)
		panicOnError(err)
		stderrStr := string(stderrBytes)

		var metrics *isolate.Metrics
		metrics, err = process.Wait()
		panicOnError(err)

		if metrics.ExitCode != 0 {
			log.Printf("Language %v failed to run. Exit code: %v. Stdout: %v. Stderr: %v", language.FullName, metrics.ExitCode, stdoutStr, stderrStr)
		} else {
			log.Printf("Language %v ran OK. Stdout: %v. Stderr: %v", language.FullName, stdoutStr, stderrStr)
		}
		// create isolate bo
		//ensureLanguageOk(language)
	}
}

func fetchLanguages() []model.ProgrammingLanguages {
	cfg := environment.ReadEnvConfig()

	log.Println("Connecting to Postgres...")
	postgres, err := sqlx.Connect("postgres", cfg.SqlxConnString)
	panicOnError(err)
	defer func(postgres *sqlx.DB) {
		err := postgres.Close()
		panicOnError(err)
	}(postgres)
	log.Println("Connected to Postgres")

	var res []model.ProgrammingLanguages

	log.Println("Selecting languages...")
	err = jet.SELECT(table.ProgrammingLanguages.AllColumns).
		FROM(table.ProgrammingLanguages).
		Query(postgres, &res)
	panicOnError(err)
	log.Printf("Selected %v languages", len(res))

	return res
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
