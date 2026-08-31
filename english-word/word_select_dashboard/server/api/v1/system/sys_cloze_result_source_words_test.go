package system

import (
	"strings"
	"testing"
	"time"
)

func sourceWordTestTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestAssembleClozeSourceWordsGameEvent(t *testing.T) {
	wrongTime := sourceWordTestTime("2026-07-26T17:01:20Z")
	wordDifficulty := 486
	answerTimeMs := int64(1200)
	item := ClozeResultItem{
		Words:                 []string{"momentum"},
		SourceEventIDs:        []int64{21},
		SourceAnswerDetailIDs: []int64{101},
		SourceRecordIDs:       []int64{11},
		SourceWordIDs:         []int64{501},
	}
	event := wrongWordEventRow{
		ID:                   21,
		Source:               "rob_english_word_back",
		SourceAnswerDetailID: 101,
		RecordID:             11,
		WordID:               501,
		Word:                 "momentum",
		WordDifficulty:       &wordDifficulty,
		SelectedMeaning:      "阻力",
		CorrectMeaning:       "动力",
		CreatedAt:            wrongTime,
	}
	game := gameAnswerSourceRow{
		DetailID:                101,
		Mode:                    "solo_training",
		TrainingDifficultyGroup: "大学英语",
		TrainingDifficultyLevel: "cet4",
		AnswerTimeMs:            &answerTimeMs,
	}

	got := assembleClozeSourceWords(
		item,
		map[int64]wrongWordEventRow{21: event},
		map[int64]gameAnswerSourceRow{101: game},
		nil,
	)

	if len(got) != 1 {
		t.Fatalf("expected one source word, got %d", len(got))
	}
	sourceWord := got[0]
	if sourceWord.Mode != "单人训练" {
		t.Fatalf("expected solo mode label, got %q", sourceWord.Mode)
	}
	if sourceWord.DifficultyGroup != "大学英语" || sourceWord.DifficultyLevel != "cet4" {
		t.Fatalf("unexpected difficulty: %#v", sourceWord)
	}
	if sourceWord.SelectedAnswer != "阻力" || sourceWord.CorrectAnswer != "动力" {
		t.Fatalf("unexpected answers: %#v", sourceWord)
	}
	if sourceWord.AnswerTimeMs == nil || *sourceWord.AnswerTimeMs != 1200 {
		t.Fatalf("unexpected answer time: %#v", sourceWord.AnswerTimeMs)
	}
	if sourceWord.WrongTime == nil || !sourceWord.WrongTime.Equal(wrongTime) {
		t.Fatalf("unexpected wrong time: %#v", sourceWord.WrongTime)
	}
	if sourceWord.TraceStatus != "available" {
		t.Fatalf("expected available trace, got %q", sourceWord.TraceStatus)
	}
	if !strings.Contains(sourceWord.TraceText, "事件 #21") ||
		!strings.Contains(sourceWord.TraceText, "答题 #101") ||
		!strings.Contains(sourceWord.TraceText, "记录 #11") {
		t.Fatalf("unexpected trace text: %q", sourceWord.TraceText)
	}
}

func TestAssembleClozeSourceWordsClozeEvent(t *testing.T) {
	eventTime := sourceWordTestTime("2026-07-26T18:02:10Z")
	answerTimeMs := int64(2350)
	item := ClozeResultItem{
		Words:                 []string{"fracture"},
		SourceEventIDs:        []int64{31},
		SourceAnswerDetailIDs: []int64{9002},
		SourceRecordIDs:       []int64{77},
	}
	event := wrongWordEventRow{
		ID:                   31,
		Source:               "sentence_cloze_practice",
		SourceAnswerDetailID: 9002,
		RecordID:             77,
		Word:                 "fracture",
		SelectedMeaning:      "破坏",
		CorrectMeaning:       "断裂",
		CreatedAt:            eventTime,
	}
	answerRecord := clozeAnswerSourceRow{
		ID:         9,
		CostMs:     &answerTimeMs,
		CreateTime: eventTime.Add(-time.Second),
	}

	got := assembleClozeSourceWords(
		item,
		map[int64]wrongWordEventRow{31: event},
		nil,
		map[int64]clozeAnswerSourceRow{9: answerRecord},
	)

	if len(got) != 1 {
		t.Fatalf("expected one source word, got %d", len(got))
	}
	sourceWord := got[0]
	if sourceWord.SourceLabel != "句子挖空练习" {
		t.Fatalf("unexpected source label: %q", sourceWord.SourceLabel)
	}
	if sourceWord.Mode != "-" {
		t.Fatalf("cloze mode must not be inferred, got %q", sourceWord.Mode)
	}
	if sourceWord.DifficultyGroup != "" || sourceWord.DifficultyLevel != "" {
		t.Fatalf("cloze difficulty must remain empty: %#v", sourceWord)
	}
	if sourceWord.AnswerTimeMs == nil || *sourceWord.AnswerTimeMs != 2350 {
		t.Fatalf("unexpected answer time: %#v", sourceWord.AnswerTimeMs)
	}
	if sourceWord.SelectedAnswer != "破坏" || sourceWord.CorrectAnswer != "fracture" {
		t.Fatalf("cloze answers must use submitted text and expected word: %#v", sourceWord)
	}
	if sourceWord.WrongTime == nil || !sourceWord.WrongTime.Equal(answerRecord.CreateTime) {
		t.Fatalf("expected answer record time, got %#v", sourceWord.WrongTime)
	}
}

func TestAssembleClozeSourceWordsMissingEvent(t *testing.T) {
	item := ClozeResultItem{
		Words:                 []string{"raw"},
		SourceEventIDs:        []int64{404},
		SourceAnswerDetailIDs: []int64{405},
		SourceRecordIDs:       []int64{406},
		SourceWordIDs:         []int64{407},
	}

	got := assembleClozeSourceWords(item, nil, nil, nil)

	if len(got) != 1 {
		t.Fatalf("expected one source word, got %d", len(got))
	}
	sourceWord := got[0]
	if sourceWord.Word != "raw" || sourceWord.SourceEventID != 404 ||
		sourceWord.SourceAnswerDetailID != 405 || sourceWord.SourceRecordID != 406 ||
		sourceWord.SourceWordID != 407 {
		t.Fatalf("missing event must preserve indexed values: %#v", sourceWord)
	}
	if sourceWord.TraceStatus != "missing" || sourceWord.TraceText != "来源事件 #404 缺失" {
		t.Fatalf("unexpected missing trace: %#v", sourceWord)
	}
}

func TestAssembleClozeSourceWordsHistoricalItem(t *testing.T) {
	item := ClozeResultItem{
		Word:  "wheel",
		Words: []string{"wheel", "stone"},
	}

	got := assembleClozeSourceWords(item, nil, nil, nil)

	if len(got) != 2 {
		t.Fatalf("expected two historical words, got %d", len(got))
	}
	for index, sourceWord := range got {
		if sourceWord.Word != item.Words[index] {
			t.Fatalf("historical order changed at %d: %#v", index, sourceWord)
		}
		if sourceWord.TraceStatus != "historical" ||
			sourceWord.TraceText != "历史生成，无答题来源" {
			t.Fatalf("unexpected historical trace: %#v", sourceWord)
		}
	}
}
