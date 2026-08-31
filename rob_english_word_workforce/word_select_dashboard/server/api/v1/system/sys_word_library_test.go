package system

import (
	"strings"
	"testing"
)

func TestWordLibraryWordsUsesExactBestSentenceJoin(t *testing.T) {
	joinSQL := wordLibraryBestSentenceJoinSQL()
	if !strings.Contains(joinSQL, "LEFT JOIN word_clean_best_sentence wcbs ON wcbs.word = w.word") {
		t.Fatalf("expected exact word join, got %q", joinSQL)
	}
	if strings.Contains(strings.ToLower(joinSQL), "lower(") {
		t.Fatalf("join must not normalize case, got %q", joinSQL)
	}

	selectSQL := wordLibraryBestSentenceSelectSQL()
	if !strings.Contains(selectSQL, "COALESCE(wcbs.sentence, '') AS sentence") {
		t.Fatalf("sentence must come from best sentence table, got %q", selectSQL)
	}
	if !strings.Contains(selectSQL, "COALESCE(wcbs.sentence_translation, '') AS sentence_translation") {
		t.Fatalf("translation must come from best sentence table, got %q", selectSQL)
	}
	for _, field := range []string{
		"COALESCE(wcbs.tts_status, '') AS best_sentence_tts_status",
		"COALESCE(wcbs.tts_bucket, '') AS best_sentence_tts_bucket",
		"COALESCE(wcbs.tts_object_key, '') AS best_sentence_tts_object_key",
		"COALESCE(wcbs.tts_object_url, '') AS best_sentence_tts_object_url",
	} {
		if !strings.Contains(selectSQL, field) {
			t.Fatalf("word list must expose best sentence TTS field %q in %q", field, selectSQL)
		}
	}
}

func TestWordLibraryWordWhereSearchesBestSentence(t *testing.T) {
	whereSQL, args := wordLibraryWordWhere(7, "harbor")
	if !strings.Contains(whereSQL, "COALESCE(wcbs.sentence, '') ILIKE ?") {
		t.Fatalf("keyword search must use best sentence, got %q", whereSQL)
	}
	if strings.Contains(whereSQL, "COALESCE(w.sentence, '') ILIKE ?") {
		t.Fatalf("keyword search must not use legacy sentence, got %q", whereSQL)
	}
	if len(args) != 5 {
		t.Fatalf("expected library id plus four keyword args, got %d", len(args))
	}
}

func TestWordUniqueIndexStatementsUseRawWord(t *testing.T) {
	statements := wordUniqueIndexStatements()
	joined := strings.Join(statements, "\n")
	for _, expected := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_word ON word_clean(word)",
		"CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_best_sentence_word ON word_clean_best_sentence(word)",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing exact unique index %q in %q", expected, joined)
		}
	}
	if strings.Contains(strings.ToLower(joined), "lower(") {
		t.Fatalf("exact unique indexes must not use lower(word), got %q", joined)
	}
}

func TestWordCleanListIncludesBaseWordTTS(t *testing.T) {
	joinSQL := wordCleanTTSJoinSQL()
	if !strings.Contains(joinSQL, "LEFT JOIN word_clean_tts wct ON wct.word_clean_id = wc.id") {
		t.Fatalf("expected word_clean_id TTS join, got %q", joinSQL)
	}
	for _, field := range []string{
		"COALESCE(wct.status, '') AS word_tts_status",
		"COALESCE(wct.tts_bucket, '') AS word_tts_bucket",
		"COALESCE(wct.tts_object_key, '') AS word_tts_object_key",
		"COALESCE(wct.tts_object_url, '') AS word_tts_object_url",
	} {
		if !strings.Contains(wordCleanTTSSelectSQL(), field) {
			t.Fatalf("missing base word TTS field %q", field)
		}
	}
}
