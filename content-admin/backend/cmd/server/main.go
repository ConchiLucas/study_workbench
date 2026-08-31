package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/conchi/study-content-admin/internal/catalog"
	"github.com/conchi/study-content-admin/internal/configclient"
	"github.com/conchi/study-content-admin/internal/db"
	"github.com/conchi/study-content-admin/internal/english"
	"github.com/conchi/study-content-admin/internal/glyph"
	httpapi "github.com/conchi/study-content-admin/internal/http"
	"github.com/conchi/study-content-admin/internal/imagegen"
	"github.com/conchi/study-content-admin/internal/literacy"
	contentmath "github.com/conchi/study-content-admin/internal/math"
	"github.com/conchi/study-content-admin/internal/pinyin"
	"github.com/conchi/study-content-admin/internal/qtask"
	"github.com/conchi/study-content-admin/internal/science"
	"github.com/conchi/study-content-admin/internal/storage"
	"github.com/conchi/study-content-admin/internal/tts"
)

func main() {
	addr := env("APP_ADDR", ":19091")
	baseURL := os.Getenv("SHARED_CONFIG_CENTER_BASE_URL")
	client := configclient.New(baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.Load(ctx); err != nil {
		log.Printf("shared config center not ready at startup: %v", err)
	}

	dsn := db.DSNFromEnv()
	gdb, err := db.OpenPostgres(dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(gdb); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	var objStore *storage.ObjectStore
	if snap, err := client.Require(context.Background()); err != nil {
		log.Printf("config snapshot unavailable for minio: %v", err)
	} else if s, err := storage.NewFromConfig(snap.ObjectStorage); err != nil {
		log.Printf("minio not ready: %v", err)
	} else {
		objStore = s
	}
	var store literacy.ObjectPutter
	if objStore != nil {
		store = objStore
	}

	var renderer literacy.GlyphRenderer
	var englishRenderer english.GlyphRenderer
	if r, err := glyph.Default(); err != nil {
		log.Printf("glyph font not ready: %v", err)
	} else {
		renderer = r
		englishRenderer = r
	}

	var pinyinRenderer pinyin.GlyphRenderer
	if r, err := glyph.Default(); err == nil {
		pinyinRenderer = r
	}

	var mathRenderer contentmath.GlyphRenderer
	if r, err := glyph.Default(); err == nil {
		mathRenderer = r
	}

	var scienceRenderer science.GlyphRenderer
	if r, err := glyph.Default(); err == nil {
		scienceRenderer = r
	}

	ttsClient := tts.New()
	router := httpapi.NewRouter(httpapi.Deps{
		Catalog:  catalog.NewService(client),
		Literacy: literacy.NewService(gdb, store, renderer, imagegen.New(), client, client, ttsClient),
		English:  english.NewService(gdb, objStore, englishRenderer, imagegen.New(), client, client, ttsClient),
		Pinyin:   pinyin.NewService(gdb, objStore, client, ttsClient, pinyinRenderer),
		Math:     contentmath.NewService(gdb, objStore, mathRenderer, client, ttsClient),
		Science:  science.NewService(gdb, objStore, scienceRenderer, imagegen.New(), client, client, ttsClient),
		QTask:    qtask.NewService(gdb),
	})
	log.Printf("study-content-admin listening on %s (config=%s db=%s)", addr, baseURL, env("APP_DB_NAME", "study_workbench"))
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
