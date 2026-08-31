package main

import (
	"log"

	"github.com/conchi/study-workbench/internal/config"
	"github.com/conchi/study-workbench/internal/db"
	apphttp "github.com/conchi/study-workbench/internal/http"
	"github.com/conchi/study-workbench/internal/repo"
	"github.com/conchi/study-workbench/internal/service"
)

func main() {
	cfg := config.Load()

	gdb, err := db.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		log.Fatal(err)
	}

	r := repo.New(gdb)
	attempts := service.NewAttemptService(r, cfg.Mastery)
	router := apphttp.NewRouter(apphttp.Deps{
		Attempt:   attempts,
		Dashboard: service.NewDashboardService(r),
		Stats:     service.NewStatsService(r),
		Reward:    service.NewRewardService(r),
		Plan:      service.NewPlanService(r, attempts),
	})

	log.Printf("db=%s driver=%s listening on %s", cfg.Pgsql.Dbname, cfg.Driver, cfg.Addr)
	if err := router.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
