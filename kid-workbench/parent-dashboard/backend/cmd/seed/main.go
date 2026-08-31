package main

import (
	"flag"
	"log"

	"github.com/conchi/study-workbench/internal/config"
	"github.com/conchi/study-workbench/internal/db"
	"github.com/conchi/study-workbench/internal/seed"
)

func main() {
	mode := flag.String("mode", "catalog", "catalog | questions | demo | recompute")
	days := flag.Int("days", 60, "demo 模式生成多少天的记录")
	flag.Parse()

	cfg := config.Load()
	gdb, err := db.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		log.Fatal(err)
	}

	switch *mode {
	case "catalog":
		if err := seed.Catalog(gdb); err != nil {
			log.Fatal(err)
		}
		log.Println("catalog seeded")
	case "questions":
		stats, err := seed.Questions(gdb)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("题库灌入完成：%d 道题，覆盖 %d 个知识点，跳过 %d 个",
			stats.Questions, stats.Kps, stats.Skipped)
		for _, code := range []string{"math", "pinyin", "literacy", "english", "science", "poem", "logic", "chengyu", "phrase"} {
			log.Printf("  %-9s %d 道", code, stats.BySubject[code])
		}
	case "demo":
		if err := seed.Demo(gdb, cfg.Mastery, 1, *days); err != nil {
			log.Fatal(err)
		}
		log.Printf("demo data seeded for %d days", *days)
	case "recompute":
		if err := seed.Recompute(gdb, cfg.Mastery, 1); err != nil {
			log.Fatal(err)
		}
		log.Println("aggregates recomputed")
	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}
