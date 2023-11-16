package main

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/programme-lv/tester/internal/environment"
	"log"
	"os/exec"
)

func main() {
	cfg := environment.ReadEnvConfig()

	log.Println("Connecting to Postgres...")
	postgres, err := sqlx.Connect("postgres", cfg.SqlxConnString)
	panicOnError(err)
	defer func(postgres *sqlx.DB) {
		err := postgres.Close()
		panicOnError(err)
	}(postgres)
	log.Println("Connected to Postgres")

	ensureIsolateOk()
}

func ensureIsolateOk() {
	isolateCmd := exec.Command("isolate", "--cg", "--cleanup")
	log.Printf("Running %v...", isolateCmd.Args)
	err := isolateCmd.Run()
	panicOnError(err)
	log.Printf("Finished %v OK", isolateCmd.Args)
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
