package system

import (
	"fmt"
	"strings"
	"time"

	"github.com/conchi/go-react-template/server/global"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	clozeTraceAvailable  = "available"
	clozeTraceHistorical = "historical"
	clozeTraceMissing    = "missing"
)

type ClozeSourceWord struct {
	Word                 string     `json:"word"`
	TraceStatus          string     `json:"traceStatus"`
	Source               string     `json:"source"`
	SourceLabel          string     `json:"sourceLabel"`
	SourceEventID        int64      `json:"sourceEventId"`
	SourceAnswerDetailID int64      `json:"sourceAnswerDetailId"`
	SourceRecordID       int64      `json:"sourceRecordId"`
	SourceWordID         int64      `json:"sourceWordId"`
	WrongTime            *time.Time `json:"wrongTime"`
	Mode                 string     `json:"mode"`
	DifficultyGroup      string     `json:"difficultyGroup"`
	DifficultyLevel      string     `json:"difficultyLevel"`
	WordDifficulty       *int       `json:"wordDifficulty"`
	AnswerTimeMs         *int64     `json:"answerTimeMs"`
	SelectedAnswer       string     `json:"selectedAnswer"`
	CorrectAnswer        string     `json:"correctAnswer"`
	TraceText            string     `json:"traceText"`
}

type wrongWordEventRow struct {
	ID                   int64     `gorm:"column:id"`
	Source               string    `gorm:"column:source"`
	SourceAnswerDetailID int64     `gorm:"column:source_answer_detail_id"`
	RecordID             int64     `gorm:"column:record_id"`
	WordID               int64     `gorm:"column:word_id"`
	Word                 string    `gorm:"column:word"`
	WordDifficulty       *int      `gorm:"column:word_difficulty"`
	SelectedMeaning      string    `gorm:"column:selected_meaning"`
	CorrectMeaning       string    `gorm:"column:correct_meaning"`
	CreatedAt            time.Time `gorm:"column:created_at"`
}

type gameAnswerSourceRow struct {
	DetailID                int64     `gorm:"column:detail_id"`
	Mode                    string    `gorm:"column:mode"`
	MatchDifficultyGroup    string    `gorm:"column:match_difficulty_group"`
	MatchDifficultyLevel    string    `gorm:"column:match_difficulty_level"`
	TrainingDifficultyGroup string    `gorm:"column:training_difficulty_group"`
	TrainingDifficultyLevel string    `gorm:"column:training_difficulty_level"`
	AnswerTimeMs            *int64    `gorm:"column:answer_time_ms"`
	CreateTime              time.Time `gorm:"column:create_time"`
}

type clozeAnswerSourceRow struct {
	ID         int64     `gorm:"column:id"`
	CostMs     *int64    `gorm:"column:cost_ms"`
	CreateTime time.Time `gorm:"column:create_time"`
}

func loadClozeSourceWords(selectDB, robDB *gorm.DB, items []ClozeResultItem) []ClozeResultItem {
	eventIDs := make([]int64, 0)
	seenEventIDs := make(map[int64]struct{})
	for _, item := range items {
		for _, eventID := range item.SourceEventIDs {
			if eventID <= 0 {
				continue
			}
			if _, exists := seenEventIDs[eventID]; exists {
				continue
			}
			seenEventIDs[eventID] = struct{}{}
			eventIDs = append(eventIDs, eventID)
		}
	}

	eventsByID := make(map[int64]wrongWordEventRow)
	if len(eventIDs) > 0 && selectDB != nil {
		var events []wrongWordEventRow
		err := selectDB.
			Table("public.wrong_word_events").
			Select("id, source, source_answer_detail_id, record_id, word_id, word, word_difficulty, selected_meaning, correct_meaning, created_at").
			Where("id IN ?", eventIDs).
			Find(&events).Error
		if err != nil {
			logClozeSourceLookupWarning("批量读取错题来源事件失败", err)
		} else {
			for _, event := range events {
				eventsByID[event.ID] = event
			}
		}
	}

	gameDetailIDs := make([]int64, 0)
	clozeAnswerRecordIDs := make([]int64, 0)
	seenGameDetailIDs := make(map[int64]struct{})
	seenClozeRecordIDs := make(map[int64]struct{})
	for _, event := range eventsByID {
		switch event.Source {
		case "rob_english_word_back":
			if event.SourceAnswerDetailID > 0 {
				if _, exists := seenGameDetailIDs[event.SourceAnswerDetailID]; !exists {
					seenGameDetailIDs[event.SourceAnswerDetailID] = struct{}{}
					gameDetailIDs = append(gameDetailIDs, event.SourceAnswerDetailID)
				}
			}
		case "sentence_cloze_practice":
			answerRecordID := clozeAnswerRecordID(event.SourceAnswerDetailID)
			if answerRecordID > 0 {
				if _, exists := seenClozeRecordIDs[answerRecordID]; !exists {
					seenClozeRecordIDs[answerRecordID] = struct{}{}
					clozeAnswerRecordIDs = append(clozeAnswerRecordIDs, answerRecordID)
				}
			}
		}
	}

	gameDetailsByID := make(map[int64]gameAnswerSourceRow)
	if len(gameDetailIDs) > 0 && robDB != nil {
		var rows []gameAnswerSourceRow
		err := robDB.Raw(`
			SELECT d.id AS detail_id,
			       COALESCE(r.mode, '') AS mode,
			       COALESCE(r.match_difficulty_group, '') AS match_difficulty_group,
			       COALESCE(r.match_difficulty_level, '') AS match_difficulty_level,
			       COALESCE(r.training_difficulty_group, '') AS training_difficulty_group,
			       COALESCE(r.training_difficulty_level, '') AS training_difficulty_level,
			       d.answer_time_ms,
			       d.create_time
			FROM game_answer_detail d
			LEFT JOIN game_record r ON r.id = d.record_id
			WHERE d.id IN ?`, gameDetailIDs).Scan(&rows).Error
		if err != nil {
			logClozeSourceLookupWarning("批量读取游戏答题来源失败", err)
		} else {
			for _, row := range rows {
				gameDetailsByID[row.DetailID] = row
			}
		}
	}

	clozeAnswersByID := make(map[int64]clozeAnswerSourceRow)
	if len(clozeAnswerRecordIDs) > 0 && robDB != nil {
		var rows []clozeAnswerSourceRow
		err := robDB.
			Table("sentence_cloze_answer_record").
			Select("id, cost_ms, create_time").
			Where("id IN ?", clozeAnswerRecordIDs).
			Find(&rows).Error
		if err != nil {
			logClozeSourceLookupWarning("批量读取挖空答题来源失败", err)
		} else {
			for _, row := range rows {
				clozeAnswersByID[row.ID] = row
			}
		}
	}

	for index := range items {
		items[index].SourceWords = assembleClozeSourceWords(
			items[index],
			eventsByID,
			gameDetailsByID,
			clozeAnswersByID,
		)
	}
	return items
}

func assembleClozeSourceWords(
	item ClozeResultItem,
	eventsByID map[int64]wrongWordEventRow,
	gameDetailsByID map[int64]gameAnswerSourceRow,
	clozeAnswersByID map[int64]clozeAnswerSourceRow,
) []ClozeSourceWord {
	words := normalizedClozeResultWords(item)
	if len(item.SourceEventIDs) == 0 {
		sourceWords := make([]ClozeSourceWord, 0, len(words))
		for _, word := range words {
			sourceWords = append(sourceWords, ClozeSourceWord{
				Word:        word,
				TraceStatus: clozeTraceHistorical,
				Source:      item.Source,
				SourceLabel: "历史生成",
				Mode:        "-",
				TraceText:   "历史生成，无答题来源",
			})
		}
		return sourceWords
	}

	slotCount := maxInt(
		len(words),
		len(item.SourceEventIDs),
		len(item.SourceAnswerDetailIDs),
		len(item.SourceRecordIDs),
		len(item.SourceWordIDs),
	)
	sourceWords := make([]ClozeSourceWord, 0, slotCount)
	for index := 0; index < slotCount; index++ {
		eventID := int64At(item.SourceEventIDs, index)
		event, found := eventsByID[eventID]
		if !found {
			sourceWords = append(sourceWords, ClozeSourceWord{
				Word:                 stringAt(words, index),
				TraceStatus:          clozeTraceMissing,
				Source:               item.Source,
				SourceLabel:          "来源记录缺失",
				SourceEventID:        eventID,
				SourceAnswerDetailID: int64At(item.SourceAnswerDetailIDs, index),
				SourceRecordID:       int64At(item.SourceRecordIDs, index),
				SourceWordID:         int64At(item.SourceWordIDs, index),
				Mode:                 "-",
				TraceText:            fmt.Sprintf("来源事件 #%d 缺失", eventID),
			})
			continue
		}

		sourceWord := ClozeSourceWord{
			Word:                 strings.TrimSpace(event.Word),
			TraceStatus:          clozeTraceAvailable,
			Source:               event.Source,
			SourceLabel:          clozeSourceLabel(event.Source),
			SourceEventID:        event.ID,
			SourceAnswerDetailID: event.SourceAnswerDetailID,
			SourceRecordID:       event.RecordID,
			SourceWordID:         event.WordID,
			Mode:                 "-",
			WordDifficulty:       event.WordDifficulty,
			SelectedAnswer:       strings.TrimSpace(event.SelectedMeaning),
			CorrectAnswer:        strings.TrimSpace(event.CorrectMeaning),
			TraceText: fmt.Sprintf(
				"事件 #%d · 答题 #%d · 记录 #%d",
				event.ID,
				event.SourceAnswerDetailID,
				event.RecordID,
			),
		}
		if sourceWord.Word == "" {
			sourceWord.Word = stringAt(words, index)
		}
		if !event.CreatedAt.IsZero() {
			wrongTime := event.CreatedAt
			sourceWord.WrongTime = &wrongTime
		}

		switch event.Source {
		case "rob_english_word_back":
			if detail, exists := gameDetailsByID[event.SourceAnswerDetailID]; exists {
				sourceWord.Mode = gameModeLabel(detail.Mode)
				sourceWord.DifficultyGroup, sourceWord.DifficultyLevel = gameDifficulty(detail)
				sourceWord.AnswerTimeMs = detail.AnswerTimeMs
				if !detail.CreateTime.IsZero() {
					wrongTime := detail.CreateTime
					sourceWord.WrongTime = &wrongTime
				}
			}
		case "sentence_cloze_practice":
			sourceWord.CorrectAnswer = sourceWord.Word
			if answer, exists := clozeAnswersByID[clozeAnswerRecordID(event.SourceAnswerDetailID)]; exists {
				sourceWord.AnswerTimeMs = answer.CostMs
				if !answer.CreateTime.IsZero() {
					wrongTime := answer.CreateTime
					sourceWord.WrongTime = &wrongTime
				}
			}
		}

		sourceWords = append(sourceWords, sourceWord)
	}
	return sourceWords
}

func normalizedClozeResultWords(item ClozeResultItem) []string {
	words := make([]string, 0, len(item.Words))
	for _, word := range item.Words {
		if cleaned := strings.TrimSpace(word); cleaned != "" {
			words = append(words, cleaned)
		}
	}
	if len(words) == 0 {
		if word := strings.TrimSpace(item.Word); word != "" {
			words = append(words, word)
		}
	}
	return words
}

func clozeAnswerRecordID(sourceAnswerDetailID int64) int64 {
	if sourceAnswerDetailID <= 0 {
		return 0
	}
	return sourceAnswerDetailID / 1000
}

func clozeSourceLabel(source string) string {
	switch source {
	case "rob_english_word_back":
		return "游戏答题"
	case "sentence_cloze_practice":
		return "句子挖空练习"
	default:
		if strings.TrimSpace(source) == "" {
			return "外部错题"
		}
		return source
	}
}

func gameModeLabel(mode string) string {
	switch mode {
	case "match":
		return "正式匹配"
	case "solo_training":
		return "单人训练"
	default:
		return "-"
	}
}

func gameDifficulty(row gameAnswerSourceRow) (string, string) {
	if row.Mode == "match" {
		return row.MatchDifficultyGroup, row.MatchDifficultyLevel
	}
	if row.Mode == "solo_training" {
		return row.TrainingDifficultyGroup, row.TrainingDifficultyLevel
	}
	return "", ""
}

func int64At(values []int64, index int) int64 {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

func stringAt(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return values[index]
}

func maxInt(values ...int) int {
	maximum := 0
	for _, value := range values {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func logClozeSourceLookupWarning(message string, err error) {
	if global.GVA_LOG != nil {
		global.GVA_LOG.Warn(message, zap.Error(err))
	}
}
