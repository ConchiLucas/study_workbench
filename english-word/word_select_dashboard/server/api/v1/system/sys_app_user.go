package system

import (
	"strconv"
	"strings"
	"time"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AppUserApi struct{}

type AppUserItem struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	Nickname           string    `json:"nickname"`
	Rank               int       `json:"rank"`
	Exp                int       `json:"exp"`
	TotalWins          int       `json:"totalWins"`
	TotalGames         int       `json:"totalGames"`
	CurrentWinStreak   int       `json:"currentWinStreak"`
	TrainingRank       int       `json:"trainingRank"`
	TrainingExp        int       `json:"trainingExp"`
	TrainingTotalWins  int       `json:"trainingTotalWins"`
	TrainingTotalGames int       `json:"trainingTotalGames"`
	CreateTime         time.Time `json:"createTime"`
	UpdateTime         time.Time `json:"updateTime"`
}

type AppUserTrainingRound struct {
	RecordID                int64                         `json:"recordId"`
	Mode                    string                        `json:"mode"`
	StartTime               time.Time                     `json:"startTime"`
	DurationSeconds         int                           `json:"durationSeconds"`
	TrainingDifficultyGroup string                        `json:"trainingDifficultyGroup"`
	TrainingDifficultyLevel string                        `json:"trainingDifficultyLevel"`
	OpponentName            string                        `json:"opponentName"`
	ResultLabel             string                        `json:"resultLabel"`
	CorrectCount            int                           `json:"correctCount"`
	TotalCount              int                           `json:"totalCount"`
	Score                   int                           `json:"score"`
	Details                 []AppUserTrainingAnswerDetail `json:"details" gorm:"-"`
}

type AppUserTrainingAnswerDetail struct {
	ID                  int64  `json:"id"`
	RoundNo             int    `json:"roundNo"`
	WordContent         string `json:"wordContent"`
	WordDifficulty      int    `json:"wordDifficulty"`
	Option1             string `json:"option1"`
	Option2             string `json:"option2"`
	Option3             string `json:"option3"`
	Option4             string `json:"option4"`
	CorrectAnswerIndex  int    `json:"correctAnswerIndex"`
	SelectedAnswerIndex *int   `json:"selectedAnswerIndex"`
	IsCorrect           int    `json:"isCorrect"`
	Score               int    `json:"score"`
	AnswerTimeMs        int    `json:"answerTimeMs"`
}

type AppUserWrongWordItem struct {
	UserID          int64                     `json:"userId"`
	UserName        string                    `json:"userName"`
	WordContent     string                    `json:"wordContent"`
	WrongCount      int                       `json:"wrongCount"`
	TotalAttempts   int                       `json:"totalAttempts"`
	AvgDifficulty   int                       `json:"avgDifficulty"`
	LastWrongTime   time.Time                 `json:"lastWrongTime"`
	LatestMode      string                    `json:"latestMode"`
	LatestGroup     string                    `json:"latestGroup"`
	LatestLevel     string                    `json:"latestLevel"`
	ReviewStatus    string                    `json:"reviewStatus"`
	ReviewStage     int                       `json:"reviewStage"`
	NextReviewTime  *time.Time                `json:"nextReviewTime"`
	RecentHistories []AppUserWrongWordHistory `json:"recentHistories" gorm:"-"`
}

type AppUserWrongWordHistory struct {
	DetailID                int64     `json:"detailId"`
	RecordID                int64     `json:"recordId"`
	StartTime               time.Time `json:"startTime"`
	Mode                    string    `json:"mode"`
	TrainingDifficultyGroup string    `json:"trainingDifficultyGroup"`
	TrainingDifficultyLevel string    `json:"trainingDifficultyLevel"`
	RoundNo                 int       `json:"roundNo"`
	WordDifficulty          int       `json:"wordDifficulty"`
	Option1                 string    `json:"option1"`
	Option2                 string    `json:"option2"`
	Option3                 string    `json:"option3"`
	Option4                 string    `json:"option4"`
	CorrectAnswerIndex      int       `json:"correctAnswerIndex"`
	SelectedAnswerIndex     *int      `json:"selectedAnswerIndex"`
	AnswerTimeMs            int       `json:"answerTimeMs"`
}

type AppUserClozeWrongItem struct {
	UserID          int64                      `json:"userId"`
	UserName        string                     `json:"userName"`
	ClozeItemID     int64                      `json:"clozeItemId"`
	Word            string                     `json:"word"`
	Words           []string                   `json:"words"`
	BlankWords      []string                   `json:"blankWords"`
	Sentence        string                     `json:"sentence"`
	TranslationZh   string                     `json:"translationZh"`
	ClozeSentence   string                     `json:"clozeSentence"`
	Source          string                     `json:"source"`
	WrongCount      int                        `json:"wrongCount"`
	TotalAttempts   int                        `json:"totalAttempts"`
	LastWrongTime   time.Time                  `json:"lastWrongTime"`
	LatestAttemptNo int                        `json:"latestAttemptNo"`
	RecentHistories []AppUserClozeWrongHistory `json:"recentHistories" gorm:"-"`
}

type AppUserClozeWrongHistory struct {
	RecordID      int64     `json:"recordId"`
	AttemptNo     int       `json:"attemptNo"`
	AnswerText    string    `json:"answerText"`
	Answers       []string  `json:"answers"`
	ExpectedWords []string  `json:"expectedWords"`
	CostMs        int64     `json:"costMs"`
	CreateTime    time.Time `json:"createTime"`
}

type AppUserMasteredWordItem struct {
	UserID           int64      `json:"userId"`
	UserName         string     `json:"userName"`
	WordID           int64      `json:"wordId"`
	WordContent      string     `json:"wordContent"`
	CorrectMeaning   string     `json:"correctMeaning"`
	Status           string     `json:"status"`
	Stage            int        `json:"stage"`
	CorrectCount     int        `json:"correctCount"`
	WordDifficulty   int        `json:"wordDifficulty"`
	LibraryName      string     `json:"libraryName"`
	LibraryMeaning   string     `json:"libraryMeaning"`
	FirstCorrectTime *time.Time `json:"firstCorrectTime"`
	Day1CorrectTime  *time.Time `json:"day1CorrectTime"`
	Day7CorrectTime  *time.Time `json:"day7CorrectTime"`
	NextReviewTime   *time.Time `json:"nextReviewTime"`
	LastCorrectTime  *time.Time `json:"lastCorrectTime"`
	MasteredTime     *time.Time `json:"masteredTime"`
}

type appUserClozeWrongItemRow struct {
	UserID          int64     `gorm:"column:user_id"`
	UserName        string    `gorm:"column:user_name"`
	ClozeItemID     int64     `gorm:"column:cloze_item_id"`
	Word            string    `gorm:"column:word"`
	WordsJSON       string    `gorm:"column:words_json"`
	BlankWordsJSON  string    `gorm:"column:blank_words_json"`
	Sentence        string    `gorm:"column:sentence"`
	TranslationZh   string    `gorm:"column:translation_zh"`
	ClozeSentence   string    `gorm:"column:cloze_sentence"`
	Source          string    `gorm:"column:source"`
	WrongCount      int       `gorm:"column:wrong_count"`
	TotalAttempts   int       `gorm:"column:total_attempts"`
	LastWrongTime   time.Time `gorm:"column:last_wrong_time"`
	LatestAttemptNo int       `gorm:"column:latest_attempt_no"`
}

type appUserClozeWrongHistoryRow struct {
	RecordID          int64     `gorm:"column:record_id"`
	AttemptNo         int       `gorm:"column:attempt_no"`
	AnswerText        string    `gorm:"column:answer_text"`
	AnswersJSON       string    `gorm:"column:answers_json"`
	ExpectedWordsJSON string    `gorm:"column:expected_words_json"`
	CostMs            int64     `gorm:"column:cost_ms"`
	CreateTime        time.Time `gorm:"column:create_time"`
}

func (a *AppUserApi) List(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	page, pageSize := parseClozePagination(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	whereSQL := ""
	args := []any{}
	if keyword != "" {
		whereSQL = " WHERE username ILIKE ? OR nickname ILIKE ? OR CAST(id AS TEXT) = ?"
		like := "%" + keyword + "%"
		args = append(args, like, like, keyword)
	}

	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM users"+whereSQL, args...).Scan(&total).Error; err != nil {
		global.GVA_LOG.Error("failed to count app users", zap.Error(err))
		response.FailWithMessage("获取用户总数失败: "+err.Error(), c)
		return
	}

	var items []AppUserItem
	queryArgs := append(args, pageSize, (page-1)*pageSize)
	querySQL := `
		SELECT
			id,
			username,
			COALESCE(nickname, '') AS nickname,
			COALESCE("rank", 0) AS rank,
			COALESCE(exp, 0) AS exp,
			COALESCE(total_wins, 0) AS total_wins,
			COALESCE(total_games, 0) AS total_games,
			COALESCE(current_win_streak, 0) AS current_win_streak,
			COALESCE(training_rank, 1) AS training_rank,
			COALESCE(training_exp, 0) AS training_exp,
			COALESCE(training_total_wins, 0) AS training_total_wins,
			COALESCE(training_total_games, 0) AS training_total_games,
			create_time,
			update_time
		FROM users` + whereSQL + `
		ORDER BY update_time DESC, id DESC
		LIMIT ? OFFSET ?`
	if err := db.Raw(querySQL, queryArgs...).Scan(&items).Error; err != nil {
		global.GVA_LOG.Error("failed to list app users", zap.Error(err))
		response.FailWithMessage("获取用户列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func (a *AppUserApi) TrainingResults(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	userID, err := strconv.ParseInt(strings.TrimSpace(c.Param("userId")), 10, 64)
	if err != nil || userID <= 0 {
		response.FailWithMessage("用户ID不正确", c)
		return
	}
	mode := strings.TrimSpace(c.Query("mode"))
	if mode != "match" {
		mode = "solo_training"
	}

	var rounds []AppUserTrainingRound
	recordsSQL := userTrainingRecordsSQL(mode)
	recordArgs := userTrainingRecordArgs(mode, userID)
	if err := db.Raw(recordsSQL, recordArgs...).Scan(&rounds).Error; err != nil {
		global.GVA_LOG.Error("failed to list user training rounds", zap.Error(err))
		response.FailWithMessage("获取用户答题轮次失败: "+err.Error(), c)
		return
	}

	visibleRounds := make([]AppUserTrainingRound, 0, len(rounds))
	for _, round := range rounds {
		var details []AppUserTrainingAnswerDetail
		detailsSQL := `
			SELECT
				id,
				round_no,
				word_content,
				COALESCE(word_difficulty, 0) AS word_difficulty,
				COALESCE(option_1, '') AS option_1,
				COALESCE(option_2, '') AS option_2,
				COALESCE(option_3, '') AS option_3,
				COALESCE(option_4, '') AS option_4,
				COALESCE(correct_answer_index, 0) AS correct_answer_index,
				selected_answer_index,
				COALESCE(is_correct, 0) AS is_correct,
				COALESCE(score, 0) AS score,
				COALESCE(answer_time_ms, 0) AS answer_time_ms
			FROM game_answer_detail
			WHERE record_id = ?
			  AND user_id = ?
			ORDER BY round_no ASC, id ASC`
		if err := db.Raw(detailsSQL, round.RecordID, userID).Scan(&details).Error; err != nil {
			global.GVA_LOG.Error("failed to list user training details", zap.Error(err))
			response.FailWithMessage("获取用户答题详情失败: "+err.Error(), c)
			return
		}
		if len(details) == 0 {
			continue
		}
		round.Details = details
		round.TotalCount = len(details)
		round.CorrectCount = 0
		round.Score = 0
		for _, detail := range details {
			if detail.IsCorrect == 1 {
				round.CorrectCount++
			}
			round.Score += detail.Score
		}
		visibleRounds = append(visibleRounds, round)
	}

	response.OkWithDetailed(visibleRounds, "获取成功", c)
}

func (a *AppUserApi) WrongWords(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	page, pageSize := parseClozePagination(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	userID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("userId")), 10, 64)
	whereSQL := " WHERE p.status <> 'completed'"
	args := []any{}
	if userID > 0 {
		whereSQL += " AND p.user_id = ?"
		args = append(args, userID)
	}
	if keyword != "" {
		whereSQL += " AND (p.word ILIKE ? OR u.username ILIKE ? OR u.nickname ILIKE ? OR CAST(u.id AS TEXT) = ?)"
		like := "%" + keyword + "%"
		args = append(args, like, like, like, keyword)
	}

	countSQL := `
		SELECT COUNT(*)
		FROM wrong_word_review_progress p
		LEFT JOIN users u ON u.id = p.user_id` + whereSQL
	var total int64
	if err := db.Raw(countSQL, args...).Scan(&total).Error; err != nil {
		global.GVA_LOG.Error("failed to count user wrong words", zap.Error(err))
		response.FailWithMessage("获取错题总数失败: "+err.Error(), c)
		return
	}

	querySQL := `
		WITH game_events AS (
			SELECT
				d.user_id,
				LOWER(BTRIM(d.word_content)) AS normalized_word,
				COALESCE(d.word_difficulty, 0) AS word_difficulty,
				COALESCE(r.mode, '') AS latest_mode,
				COALESCE(r.training_difficulty_group, '') AS latest_group,
				COALESCE(r.training_difficulty_level, '') AS latest_level,
				COALESCE(d.create_time, r.start_time, CURRENT_TIMESTAMP) AS event_time,
				CONCAT('game:', d.id::text) AS event_key
			FROM game_answer_detail d
			LEFT JOIN game_record r ON r.id = d.record_id
			WHERE COALESCE(d.is_correct, 0) = 0
			  AND COALESCE(BTRIM(d.word_content), '') <> ''
		),
		cloze_payload AS (
			SELECT
				r.id,
				r.user_id,
				r.create_time,
				public.wrong_word_review_safe_jsonb_array(r.answers_json) AS answers,
				public.wrong_word_review_safe_jsonb_array(r.expected_words_json) AS expected_words
			FROM sentence_cloze_answer_record r
			WHERE COALESCE(r.is_correct, false) = false
		),
		cloze_events AS (
			SELECT
				payload.user_id,
				LOWER(BTRIM(expected.word)) AS normalized_word,
				COALESCE((
					SELECT w.difficulty
					FROM word w
					WHERE LOWER(BTRIM(w.word)) = LOWER(BTRIM(expected.word))
					ORDER BY CASE WHEN w.status = 1 THEN 0 ELSE 1 END, w.id
					LIMIT 1
				), 0) AS word_difficulty,
				'cloze_review'::text AS latest_mode,
				''::text AS latest_group,
				''::text AS latest_level,
				COALESCE(payload.create_time, CURRENT_TIMESTAMP) AS event_time,
				CONCAT('cloze:', payload.id::text, ':', expected.ordinal::text) AS event_key
			FROM cloze_payload payload
			CROSS JOIN LATERAL jsonb_array_elements_text(payload.expected_words)
				WITH ORDINALITY AS expected(word, ordinal)
			LEFT JOIN LATERAL jsonb_array_elements_text(payload.answers)
				WITH ORDINALITY AS answer(word, ordinal)
				ON answer.ordinal = expected.ordinal
			WHERE BTRIM(expected.word) <> ''
			  AND (
				jsonb_array_length(payload.answers) <> jsonb_array_length(payload.expected_words)
				OR LOWER(BTRIM(COALESCE(answer.word, ''))) <> LOWER(BTRIM(expected.word))
			  )
		),
		all_events AS (
			SELECT * FROM game_events
			UNION ALL
			SELECT * FROM cloze_events
		),
		latest_event AS (
			SELECT
				all_events.*,
				ROW_NUMBER() OVER (
					PARTITION BY user_id, normalized_word
					ORDER BY event_time DESC, event_key DESC
				) AS row_no
			FROM all_events
		),
		attempts AS (
			SELECT user_id, LOWER(BTRIM(word_content)) AS normalized_word, COUNT(*) AS total_attempts
			FROM game_answer_detail
			WHERE COALESCE(BTRIM(word_content), '') <> ''
			GROUP BY user_id, LOWER(BTRIM(word_content))
		)
		SELECT
			p.user_id,
			COALESCE(NULLIF(u.nickname, ''), NULLIF(u.username, ''), CONCAT('用户 ', p.user_id::text)) AS user_name,
			p.word AS word_content,
			p.wrong_count,
			GREATEST(COALESCE(a.total_attempts, 0), p.wrong_count) AS total_attempts,
			COALESCE(e.word_difficulty, w.difficulty, 0)::int AS avg_difficulty,
			p.last_wrong_time,
			COALESCE(NULLIF(e.latest_mode, ''), CASE WHEN p.active_cloze_item_id IS NOT NULL THEN 'cloze_review' ELSE '' END) AS latest_mode,
			COALESCE(e.latest_group, '') AS latest_group,
			COALESCE(e.latest_level, '') AS latest_level,
			p.status AS review_status,
			p.review_stage,
			p.next_review_time
		FROM wrong_word_review_progress p
		LEFT JOIN users u ON u.id = p.user_id
		LEFT JOIN word w ON w.id = p.word_id
		LEFT JOIN latest_event e
			ON e.user_id = p.user_id
			AND e.normalized_word = p.normalized_word
			AND e.row_no = 1
		LEFT JOIN attempts a
			ON a.user_id = p.user_id
			AND a.normalized_word = p.normalized_word` + whereSQL + `
		ORDER BY p.last_wrong_time DESC, p.wrong_count DESC, p.id DESC
		LIMIT ? OFFSET ?`
	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)

	var items []AppUserWrongWordItem
	if err := db.Raw(querySQL, queryArgs...).Scan(&items).Error; err != nil {
		global.GVA_LOG.Error("failed to list user wrong words", zap.Error(err))
		response.FailWithMessage("获取错题列表失败: "+err.Error(), c)
		return
	}

	for index := range items {
		historiesSQL := `
			SELECT
				d.id AS detail_id,
				d.record_id,
				COALESCE(d.create_time, r.start_time, CURRENT_TIMESTAMP) AS start_time,
				COALESCE(r.mode, '') AS mode,
				COALESCE(r.training_difficulty_group, '') AS training_difficulty_group,
				COALESCE(r.training_difficulty_level, '') AS training_difficulty_level,
				COALESCE(d.round_no, 0) AS round_no,
				COALESCE(d.word_difficulty, 0) AS word_difficulty,
				COALESCE(d.option_1, '') AS option_1,
				COALESCE(d.option_2, '') AS option_2,
				COALESCE(d.option_3, '') AS option_3,
				COALESCE(d.option_4, '') AS option_4,
				COALESCE(d.correct_answer_index, 0) AS correct_answer_index,
				d.selected_answer_index,
				COALESCE(d.answer_time_ms, 0) AS answer_time_ms
			FROM game_answer_detail d
			LEFT JOIN game_record r ON r.id = d.record_id
			WHERE d.user_id = ?
			  AND LOWER(BTRIM(d.word_content)) = LOWER(BTRIM(?))
			  AND COALESCE(d.is_correct, 0) = 0
			ORDER BY COALESCE(d.create_time, r.start_time, CURRENT_TIMESTAMP) DESC, d.id DESC
			LIMIT 20`
		var histories []AppUserWrongWordHistory
		if err := db.Raw(historiesSQL, items[index].UserID, items[index].WordContent).Scan(&histories).Error; err != nil {
			global.GVA_LOG.Error("failed to list user wrong word histories", zap.Error(err))
			response.FailWithMessage("获取错题明细失败: "+err.Error(), c)
			return
		}
		items[index].RecentHistories = histories
	}

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func (a *AppUserApi) ClozeWrongWords(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	page, pageSize := parseClozePagination(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	userID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("userId")), 10, 64)
	whereSQL := " WHERE COALESCE(r.is_correct, false) = false AND r.cloze_item_id > 0"
	args := []any{}
	if userID > 0 {
		whereSQL += " AND r.user_id = ?"
		args = append(args, userID)
	}
	if keyword != "" {
		whereSQL += ` AND (
			i.word ILIKE ? OR i.sentence ILIKE ? OR i.cloze_sentence ILIKE ? OR
			i.translation_zh ILIKE ? OR u.username ILIKE ? OR u.nickname ILIKE ? OR
			CAST(u.id AS TEXT) LIKE ?
		)`
		like := "%" + keyword + "%"
		args = append(args, like, like, like, like, like, like, like)
	}

	countSQL := `
		SELECT COUNT(*)
		FROM (
			SELECT r.user_id, r.cloze_item_id
			FROM sentence_cloze_answer_record r
			LEFT JOIN sentence_cloze_item i ON i.id = r.cloze_item_id
			LEFT JOIN users u ON u.id = r.user_id` + whereSQL + `
			GROUP BY r.user_id, r.cloze_item_id
		) wrong_cloze`
	var total int64
	if err := db.Raw(countSQL, args...).Scan(&total).Error; err != nil {
		global.GVA_LOG.Error("failed to count user cloze wrong words", zap.Error(err))
		response.FailWithMessage("获取造句错题总数失败: "+err.Error(), c)
		return
	}

	querySQL := `
		WITH wrong_group AS (
			SELECT
				r.user_id,
				COALESCE(NULLIF(u.nickname, ''), NULLIF(u.username, ''), NULLIF(r.user_name, ''), CONCAT('用户 ', r.user_id::text)) AS user_name,
				r.cloze_item_id,
				COALESCE(i.word, '') AS word,
				COALESCE(i.words_json, '[]') AS words_json,
				COALESCE(i.blank_words_json, '[]') AS blank_words_json,
				COALESCE(i.sentence, '') AS sentence,
				COALESCE(i.translation_zh, '') AS translation_zh,
				COALESCE(i.cloze_sentence, '') AS cloze_sentence,
				COALESCE(i.source, '') AS source,
				COUNT(*) AS wrong_count,
				MAX(COALESCE(r.create_time, CURRENT_TIMESTAMP)) AS last_wrong_time
			FROM sentence_cloze_answer_record r
			LEFT JOIN sentence_cloze_item i ON i.id = r.cloze_item_id
			LEFT JOIN users u ON u.id = r.user_id` + whereSQL + `
			GROUP BY
				r.user_id, u.nickname, u.username, r.user_name, r.cloze_item_id,
				i.word, i.words_json, i.blank_words_json, i.sentence, i.translation_zh,
				i.cloze_sentence, i.source
		),
		latest_wrong AS (
			SELECT
				r.user_id,
				r.cloze_item_id,
				COALESCE(r.attempt_no, 1) AS latest_attempt_no,
				ROW_NUMBER() OVER (
					PARTITION BY r.user_id, r.cloze_item_id
					ORDER BY COALESCE(r.create_time, CURRENT_TIMESTAMP) DESC, r.id DESC
				) AS row_no
			FROM sentence_cloze_answer_record r
			LEFT JOIN sentence_cloze_item i ON i.id = r.cloze_item_id
			LEFT JOIN users u ON u.id = r.user_id` + whereSQL + `
		),
		attempts AS (
			SELECT user_id, cloze_item_id, COUNT(*) AS total_attempts
			FROM sentence_cloze_answer_record
			WHERE cloze_item_id > 0
			GROUP BY user_id, cloze_item_id
		)
		SELECT
			g.user_id,
			g.user_name,
			g.cloze_item_id,
			g.word,
			g.words_json,
			g.blank_words_json,
			g.sentence,
			g.translation_zh,
			g.cloze_sentence,
			g.source,
			g.wrong_count,
			COALESCE(a.total_attempts, g.wrong_count) AS total_attempts,
			g.last_wrong_time,
			COALESCE(l.latest_attempt_no, 1) AS latest_attempt_no
		FROM wrong_group g
		LEFT JOIN latest_wrong l ON l.user_id = g.user_id AND l.cloze_item_id = g.cloze_item_id AND l.row_no = 1
		LEFT JOIN attempts a ON a.user_id = g.user_id AND a.cloze_item_id = g.cloze_item_id
		ORDER BY g.last_wrong_time DESC, g.wrong_count DESC, g.user_id DESC
		LIMIT ? OFFSET ?`
	queryArgs := append(append([]any{}, args...), args...)
	queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)

	var rows []appUserClozeWrongItemRow
	if err := db.Raw(querySQL, queryArgs...).Scan(&rows).Error; err != nil {
		global.GVA_LOG.Error("failed to list user cloze wrong words", zap.Error(err))
		response.FailWithMessage("获取造句错题列表失败: "+err.Error(), c)
		return
	}

	items := make([]AppUserClozeWrongItem, 0, len(rows))
	for _, row := range rows {
		historiesSQL := `
			SELECT
				r.id AS record_id,
				COALESCE(r.attempt_no, 1) AS attempt_no,
				COALESCE(r.answer_text, '') AS answer_text,
				COALESCE(r.answers_json, '[]') AS answers_json,
				COALESCE(r.expected_words_json, '[]') AS expected_words_json,
				COALESCE(r.cost_ms, 0) AS cost_ms,
				COALESCE(r.create_time, CURRENT_TIMESTAMP) AS create_time
			FROM sentence_cloze_answer_record r
			WHERE r.user_id = ?
			  AND r.cloze_item_id = ?
			  AND COALESCE(r.is_correct, false) = false
			ORDER BY COALESCE(r.create_time, CURRENT_TIMESTAMP) DESC, r.id DESC
			LIMIT 20`
		var historyRows []appUserClozeWrongHistoryRow
		if err := db.Raw(historiesSQL, row.UserID, row.ClozeItemID).Scan(&historyRows).Error; err != nil {
			global.GVA_LOG.Error("failed to list user cloze wrong histories", zap.Error(err))
			response.FailWithMessage("获取造句错题明细失败: "+err.Error(), c)
			return
		}

		histories := make([]AppUserClozeWrongHistory, 0, len(historyRows))
		for _, history := range historyRows {
			histories = append(histories, AppUserClozeWrongHistory{
				RecordID:      history.RecordID,
				AttemptNo:     history.AttemptNo,
				AnswerText:    history.AnswerText,
				Answers:       decodeStringJSONList(history.AnswersJSON),
				ExpectedWords: decodeStringJSONList(history.ExpectedWordsJSON),
				CostMs:        history.CostMs,
				CreateTime:    history.CreateTime,
			})
		}

		items = append(items, AppUserClozeWrongItem{
			UserID:          row.UserID,
			UserName:        row.UserName,
			ClozeItemID:     row.ClozeItemID,
			Word:            row.Word,
			Words:           decodeStringJSONList(row.WordsJSON),
			BlankWords:      decodeStringJSONList(row.BlankWordsJSON),
			Sentence:        row.Sentence,
			TranslationZh:   row.TranslationZh,
			ClozeSentence:   row.ClozeSentence,
			Source:          row.Source,
			WrongCount:      row.WrongCount,
			TotalAttempts:   row.TotalAttempts,
			LastWrongTime:   row.LastWrongTime,
			LatestAttemptNo: row.LatestAttemptNo,
			RecentHistories: histories,
		})
	}

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func (a *AppUserApi) MasteredWords(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	page, pageSize := parseClozePagination(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	status := strings.TrimSpace(c.Query("status"))
	if status != "learning" && status != "mastered" {
		status = ""
	}
	userID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("userId")), 10, 64)

	whereSQL := " WHERE 1=1"
	args := []any{}
	if userID > 0 {
		whereSQL += " AND p.user_id = ?"
		args = append(args, userID)
	}
	if status != "" {
		whereSQL += " AND p.status = ?"
		args = append(args, status)
	}
	if keyword != "" {
		whereSQL += ` AND (
			COALESCE(NULLIF(p.word_content, ''), w.word, '') ILIKE ? OR
			COALESCE(NULLIF(p.correct_meaning, ''), w.meaning, '') ILIKE ? OR
			u.username ILIKE ? OR
			u.nickname ILIKE ? OR
			CAST(u.id AS TEXT) = ?
		)`
		like := "%" + keyword + "%"
		args = append(args, like, like, like, like, keyword)
	}

	countSQL := `
		SELECT COUNT(*)
		FROM user_word_mastery_progress p
		LEFT JOIN word w ON w.id = p.word_id
		LEFT JOIN users u ON u.id = p.user_id` + whereSQL
	var total int64
	if err := db.Raw(countSQL, args...).Scan(&total).Error; err != nil {
		global.GVA_LOG.Error("failed to count user mastered words", zap.Error(err))
		response.FailWithMessage("获取已掌握单词总数失败: "+err.Error(), c)
		return
	}

	querySQL := `
		SELECT
			p.user_id,
			COALESCE(NULLIF(u.nickname, ''), NULLIF(u.username, ''), CONCAT('用户 ', p.user_id::text)) AS user_name,
			p.word_id,
			COALESCE(NULLIF(p.word_content, ''), w.word, '') AS word_content,
			COALESCE(NULLIF(p.correct_meaning, ''), w.meaning, '') AS correct_meaning,
			COALESCE(p.status, 'learning') AS status,
			COALESCE(p.stage, 0) AS stage,
			COALESCE(p.correct_count, 0) AS correct_count,
			COALESCE(w.difficulty, 0) AS word_difficulty,
			COALESCE(wl.library_name, '') AS library_name,
			COALESCE(wl.library_meaning, '') AS library_meaning,
			p.first_correct_time,
			p.day1_correct_time,
			p.day7_correct_time,
			p.next_review_time,
			p.last_correct_time,
			p.mastered_time
		FROM user_word_mastery_progress p
		LEFT JOIN word w ON w.id = p.word_id
		LEFT JOIN word_library wl ON wl.id = w.library_id
		LEFT JOIN users u ON u.id = p.user_id` + whereSQL + `
		ORDER BY
			CASE WHEN p.status = 'mastered' THEN 0 ELSE 1 END,
			COALESCE(p.mastered_time, p.last_correct_time, p.update_time, p.create_time) DESC,
			p.id DESC
		LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, (page-1)*pageSize)

	var items []AppUserMasteredWordItem
	if err := db.Raw(querySQL, queryArgs...).Scan(&items).Error; err != nil {
		global.GVA_LOG.Error("failed to list user mastered words", zap.Error(err))
		response.FailWithMessage("获取已掌握单词列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func userTrainingRecordsSQL(mode string) string {
	baseSelect := `
		SELECT
			id AS record_id,
			mode,
			start_time,
			COALESCE(duration_seconds, 0) AS duration_seconds,
			COALESCE(training_difficulty_group, '') AS training_difficulty_group,
			COALESCE(training_difficulty_level, '') AS training_difficulty_level,`

	if mode == "match" {
		return baseSelect + `
			CASE WHEN player1_id = ? THEN COALESCE(player2_name, '') ELSE COALESCE(player1_name, '') END AS opponent_name,
			CASE
				WHEN COALESCE(is_draw, 0) = 1 THEN '平局'
				WHEN winner_id = ? THEN '胜利'
				ELSE '失败'
			END AS result_label,
			CASE WHEN player1_id = ? THEN COALESCE(player1_correct_count, 0) ELSE COALESCE(player2_correct_count, 0) END AS correct_count,
			CASE WHEN player1_id = ? THEN COALESCE(player1_total_count, 0) ELSE COALESCE(player2_total_count, 0) END AS total_count,
			CASE WHEN player1_id = ? THEN COALESCE(player1_score, 0) ELSE COALESCE(player2_score, 0) END AS score
		FROM game_record
		WHERE mode = 'match'
		  AND (player1_id = ? OR player2_id = ?)
		  AND start_time::date = CURRENT_DATE
		ORDER BY start_time DESC, id DESC
		LIMIT 100`
	}

	return baseSelect + `
			COALESCE(player2_name, '训练机器人') AS opponent_name,
			'' AS result_label,
			COALESCE(player1_correct_count, 0) AS correct_count,
			COALESCE(player1_total_count, 0) AS total_count,
			COALESCE(player1_score, 0) AS score
		FROM game_record
		WHERE mode = 'solo_training'
		  AND player1_id = ?
		  AND start_time::date = CURRENT_DATE
		ORDER BY start_time DESC, id DESC
		LIMIT 100`
}

func userTrainingRecordArgs(mode string, userID int64) []any {
	if mode == "match" {
		return []any{userID, userID, userID, userID, userID, userID, userID}
	}
	return []any{userID}
}
