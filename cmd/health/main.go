package main

import (
	"errors"
	jet "github.com/go-jet/jet/v2/postgres"
	pretty_table "github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/programme-lv/tester/internal/database/proglv/public/model"
	"github.com/programme-lv/tester/internal/database/proglv/public/table"
	"github.com/programme-lv/tester/internal/environment"
	"github.com/programme-lv/tester/internal/isolate"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type feedbackRow struct {
	unit    string
	health  int // 0 - OK, 1 - Warning, 2 - Error
	message string
}

func main() {
	feedback := make([]feedbackRow, 0)

	isolateRow := ensureIsolateOk()
	feedback = append(feedback, isolateRow)

	if isolateRow.health != 2 {
		langRows := ensureLanguagesOk()
		feedback = append(feedback, langRows...)
	}

	outputFeedback(feedback)
}

func ensureIsolateOk() feedbackRow {
	isolateCmd := exec.Command("isolate", "--cg", "--cleanup")
	log.Printf("Running %v...", isolateCmd.Args)
	out, err := isolateCmd.CombinedOutput()
	if err != nil {
		var exitError *exec.ExitError
		ok := errors.As(err, &exitError)
		if !ok {
			return feedbackRow{
				unit:    "Isolate",
				health:  2,
				message: err.Error(),
			}
		}
	}

	log.Printf("Ran %v", isolateCmd.Args)
	if err != nil {
		log.Printf("Failed to run %v: %v", isolateCmd.Args, err)
		msg := err.Error()
		if out != nil && len(out) > 0 {
			msg = msg + ": " + string(out)
		}
		return feedbackRow{
			unit:    "Isolate",
			health:  2,
			message: msg,
		}
	} else {
		return feedbackRow{
			unit:    "Isolate",
			health:  0,
			message: string(out),
		}
	}
}

func ensureLanguagesOk() []feedbackRow {
	languages := fetchLanguages()

	isolateInstance := isolate.GetInstance()

	res := make([]feedbackRow, 0)
	for _, language := range languages {
		box, err := isolateInstance.NewBox()
		panicOnError(err)

		err = box.AddFile(language.CodeFilename, []byte(*language.HelloWorldCode))
		panicOnError(err)

		var process *isolate.Process
		process, err = box.Run(language.ExecuteCmd, io.NopCloser(strings.NewReader("")), nil)

		var metrics *isolate.Metrics
		var out []byte
		metrics, out, err = process.CombinedOutput()
		panicOnError(err)

		healthInt := 0
		if metrics.ExitCode != 0 {
			healthInt = 2
		}

		res = append(res, feedbackRow{
			unit:    language.FullName,
			health:  healthInt,
			message: string(out),
		})
	}
	return nil
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

func outputFeedback(feedback []feedbackRow) {
	t := pretty_table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(pretty_table.Row{"Unit", "Health", "Message"})
	for _, row := range feedback {
		healthCode := ""
		switch row.health {
		case 0:
			healthCode = "OKAY"
		case 1:
			healthCode = "WARN"
		case 2:
			healthCode = "ERROR"
		}

		t.AppendRow(
			pretty_table.Row{
				row.unit,
				healthCode,
				row.message,
			})
	}
	t.SetStyle(pretty_table.StyleColoredDark)
	textColor := text.Transformer(func(s interface{}) string {
		switch s.(string) {
		case "OKAY":
			return text.FgHiGreen.Sprint(s)
		case "WARN":
			return text.FgHiYellow.Sprint(s)
		case "ERROR":
			return text.FgHiRed.Sprint(s)
		}
		return ""
	})

	t.SetColumnConfigs([]pretty_table.ColumnConfig{
		{
			Name:        "Health",
			Transformer: textColor,
			Align:       text.AlignCenter,
		},
	})
	t.Render()
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
