package main

import (
	"flag"
	"log/slog"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/infra/postgres"
)

func main() {

	cnf := config.GetConfig()

	setupUser := flag.Bool("setup", false, "setup")
	up := flag.Bool("up", false, "up")
	down := flag.Bool("down", false, "down")

	flag.Parse()

	if *setupUser == true {
		if err := postgres.SetupDatabase(cnf.PostgreSQL); err != nil {
			panic(err)
		}
		slog.Info("Setup database user successfully")
	}

	if *up == true {
		if err := postgres.Run(cnf.PostgreSQL); err != nil {
			panic(err)
		}
		slog.Info("Migration success")
	}

	if *down == true {
		if err := postgres.Rollback(cnf.PostgreSQL, 1); err != nil {
			panic(err)
		}
		slog.Info("Rollback success")
	}
}
