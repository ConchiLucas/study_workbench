package seed_test

import (
	"encoding/json"
	"testing"

	"github.com/conchi/study-workbench/internal/db"
	"github.com/conchi/study-workbench/internal/seed"
)

func TestQuestionsCoverEnabledSubjectsOnly(t *testing.T) {
	gdb, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatal(err)
	}
	if err := seed.Catalog(gdb); err != nil {
		t.Fatal(err)
	}

	stats, err := seed.Questions(gdb)
	if err != nil {
		t.Fatal(err)
	}

	// 全库 1236 KP − 游戏 8 = 1228 可出题
	if stats.Kps != 1228 {
		t.Errorf("可出题知识点 = %d，期望 1228", stats.Kps)
	}
	if stats.Questions < 1100 {
		t.Errorf("题目数 = %d，太少了", stats.Questions)
	}
	for _, code := range []string{"math", "pinyin", "literacy", "english", "science", "poem", "logic", "chengyu", "phrase"} {
		if stats.BySubject[code] == 0 {
			t.Errorf("%s 没有生成任何题目", code)
		}
	}

	// 游戏科仍未开启，一道题都不该有。
	var leaked int64
	gdb.Raw(`SELECT COUNT(1) FROM questions q
		JOIN knowledge_points kp ON kp.id = q.kp_id
		JOIN modules m  ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.quiz_enabled = ?`, false).Scan(&leaked)
	if leaked != 0 {
		t.Errorf("未开启出题的学科出现了 %d 道题", leaked)
	}
}

func TestQuestionsAreIdempotent(t *testing.T) {
	gdb, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatal(err)
	}
	if err := seed.Catalog(gdb); err != nil {
		t.Fatal(err)
	}

	first, err := seed.Questions(gdb)
	if err != nil {
		t.Fatal(err)
	}
	var afterFirst int64
	gdb.Raw(`SELECT COUNT(1) FROM questions`).Scan(&afterFirst)

	if _, err := seed.Questions(gdb); err != nil {
		t.Fatal(err)
	}
	var afterSecond int64
	gdb.Raw(`SELECT COUNT(1) FROM questions`).Scan(&afterSecond)

	if afterFirst != afterSecond {
		t.Errorf("重复灌库产生了新记录：%d → %d", afterFirst, afterSecond)
	}
	if afterFirst != int64(first.Questions) {
		t.Errorf("入库数 %d 与统计 %d 不一致", afterFirst, first.Questions)
	}
}

func TestStoredQuestionsAreAnswerable(t *testing.T) {
	gdb, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatal(err)
	}
	if err := seed.Catalog(gdb); err != nil {
		t.Fatal(err)
	}
	if _, err := seed.Questions(gdb); err != nil {
		t.Fatal(err)
	}

	type row struct {
		ID      int64
		Stem    string
		Options string
		Answer  string
	}
	var rows []row
	if err := gdb.Raw(`SELECT id, stem, options, answer FROM questions`).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		var opts []map[string]string
		if err := json.Unmarshal([]byte(r.Options), &opts); err != nil {
			t.Fatalf("题 %d 选项不是合法 JSON: %v", r.ID, err)
		}
		if len(opts) != 4 {
			t.Fatalf("题 %d 有 %d 个选项", r.ID, len(opts))
		}
		var ans struct{ Index int }
		if err := json.Unmarshal([]byte(r.Answer), &ans); err != nil {
			t.Fatalf("题 %d 答案不是合法 JSON: %v", r.ID, err)
		}
		if ans.Index < 0 || ans.Index >= len(opts) {
			t.Fatalf("题 %d 正确项下标 %d 越界", r.ID, ans.Index)
		}
		if r.Stem == "" {
			t.Fatalf("题 %d 题干为空", r.ID)
		}
	}
}
