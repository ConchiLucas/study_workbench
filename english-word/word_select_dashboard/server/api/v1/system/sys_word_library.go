package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WordLibraryApi struct{}

type WordLibraryItem struct {
	ID             int64     `json:"id"`
	LibraryName    string    `json:"libraryName"`
	LibraryMeaning string    `json:"libraryMeaning"`
	Status         int       `json:"status"`
	WordCount      int       `json:"wordCount"`
	CreatedBy      *int64    `json:"createdBy"`
	CreateTime     time.Time `json:"createTime"`
	UpdateTime     time.Time `json:"updateTime"`
}

type WordLibraryWordItem struct {
	ID                       int64     `json:"id"`
	LibraryID                int64     `json:"libraryId"`
	Word                     string    `json:"word"`
	Meaning                  string    `json:"meaning"`
	PronunciationUs          string    `json:"pronunciationUs"`
	PronunciationUk          string    `json:"pronunciationUk"`
	Frequency                int       `json:"frequency"`
	Difficulty               int       `json:"difficulty"`
	Status                   int       `json:"status"`
	Phrase                   string    `json:"phrase"`
	PhraseTranslation        string    `json:"phraseTranslation"`
	Sentence                 string    `json:"sentence"`
	SentenceTranslation      string    `json:"sentenceTranslation"`
	BestSentenceTTSStatus    string    `json:"bestSentenceTtsStatus"`
	BestSentenceTTSBucket    string    `json:"bestSentenceTtsBucket"`
	BestSentenceTTSObjectKey string    `json:"bestSentenceTtsObjectKey"`
	BestSentenceTTSObjectURL string    `json:"bestSentenceTtsObjectUrl"`
	CreateTime               time.Time `json:"createTime"`
	UpdateTime               time.Time `json:"updateTime"`
}

type WordCleanItem struct {
	ID                          int64      `json:"id"`
	Word                        string     `json:"word"`
	Meaning                     string     `json:"meaning"`
	Difficulty                  int        `json:"difficulty"`
	Frequency                   int        `json:"frequency"`
	Sentence                    string     `json:"sentence"`
	PepDifficulty               *int       `json:"pepDifficulty"`
	PepDifficultyLabel          string     `json:"pepDifficultyLabel"`
	SourceDifficulty            *int       `json:"sourceDifficulty"`
	SourceLabel                 string     `json:"sourceLabel"`
	BestSentenceID              *int64     `json:"bestSentenceId"`
	BestSourceSentenceID        *int64     `json:"bestSourceSentenceId"`
	BestSourceModelName         string     `json:"bestSourceModelName"`
	BestSentence                string     `json:"bestSentence"`
	BestSentenceTranslation     string     `json:"bestSentenceTranslation"`
	BestSentenceScore           *int       `json:"bestSentenceScore"`
	BestSentenceScoreReason     string     `json:"bestSentenceScoreReason"`
	BestSentenceScoreModelName  string     `json:"bestSentenceScoreModelName"`
	BestSentenceScoredAt        *time.Time `json:"bestSentenceScoredAt"`
	BestSentenceTTSStatus       string     `json:"bestSentenceTtsStatus"`
	BestSentenceTTSBucket       string     `json:"bestSentenceTtsBucket"`
	BestSentenceTTSObjectKey    string     `json:"bestSentenceTtsObjectKey"`
	BestSentenceTTSObjectURL    string     `json:"bestSentenceTtsObjectUrl"`
	BestSentenceTTSContentType  string     `json:"bestSentenceTtsContentType"`
	BestSentenceTTSFileSize     *int64     `json:"bestSentenceTtsFileSize"`
	BestSentenceTTSDurationMs   *int       `json:"bestSentenceTtsDurationMs"`
	BestSentenceTTSGeneratedAt  *time.Time `json:"bestSentenceTtsGeneratedAt"`
	BestSentenceTTSErrorMessage string     `json:"bestSentenceTtsErrorMessage"`
	WordTTSStatus               string     `json:"wordTtsStatus"`
	WordTTSBucket               string     `json:"wordTtsBucket"`
	WordTTSObjectKey            string     `json:"wordTtsObjectKey"`
	WordTTSObjectURL            string     `json:"wordTtsObjectUrl"`
}

type WordCleanSentenceItem struct {
	ID                  int64      `json:"id"`
	WordCleanID         int64      `json:"wordCleanId"`
	Word                string     `json:"word"`
	ModelName           string     `json:"modelName"`
	Sentence            string     `json:"sentence"`
	SentenceTranslation string     `json:"sentenceTranslation"`
	Score               *int       `json:"score"`
	ScoreReason         string     `json:"scoreReason"`
	ScoreModelName      string     `json:"scoreModelName"`
	ScoredAt            *time.Time `json:"scoredAt"`
}

type ScoreWordCleanSentencesRequest struct {
	IDs          []int64  `json:"ids"`
	WordCleanIDs []int64  `json:"wordCleanIds"`
	ModelNames   []string `json:"modelNames"`
	JudgeModel   string   `json:"judgeModel"`
	Limit        int      `json:"limit"`
	Overwrite    bool     `json:"overwrite"`
}

type ScoreWordCleanSentenceItem struct {
	ID          int64  `json:"id"`
	WordCleanID int64  `json:"wordCleanId"`
	Word        string `json:"word"`
	ModelName   string `json:"modelName"`
	Score       int    `json:"score"`
	ScoreReason string `json:"scoreReason"`
}

type ScoreWordCleanBestSentenceItem struct {
	ID                  int64      `json:"id"`
	WordCleanID         int64      `json:"wordCleanId"`
	Word                string     `json:"word"`
	Meaning             string     `json:"meaning"`
	SourceSentenceID    int64      `json:"sourceSentenceId"`
	SourceModelName     string     `json:"sourceModelName"`
	Sentence            string     `json:"sentence"`
	SentenceTranslation string     `json:"sentenceTranslation"`
	Score               int        `json:"score"`
	ScoreReason         string     `json:"scoreReason"`
	ScoreModelName      string     `json:"scoreModelName"`
	ScoredAt            *time.Time `json:"scoredAt"`
	TTSStatus           string     `json:"ttsStatus"`
	TTSBucket           string     `json:"ttsBucket"`
	TTSObjectKey        string     `json:"ttsObjectKey"`
	TTSObjectURL        string     `json:"ttsObjectUrl"`
}

type ScoreWordCleanSentencesResponse struct {
	Status         string                           `json:"status"`
	Message        string                           `json:"message"`
	JudgeModel     string                           `json:"judgeModel"`
	ProcessedCount int                              `json:"processedCount"`
	ScoredCount    int                              `json:"scoredCount"`
	FailedCount    int                              `json:"failedCount"`
	Items          []ScoreWordCleanSentenceItem     `json:"items"`
	BestItems      []ScoreWordCleanBestSentenceItem `json:"bestItems"`
}

func (a *WordLibraryApi) Libraries(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	page, pageSize := parseClozePagination(c)
	if ps, err := strconv.Atoi(c.Query("pageSize")); err == nil && ps > 0 {
		pageSize = ps
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	whereSQL, args := wordLibraryWhere(keyword)

	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM word_library wl WHERE 1=1"+whereSQL, args...).Scan(&total).Error; err != nil {
		response.FailWithMessage("获取词库总数失败: "+err.Error(), c)
		return
	}

	var items []WordLibraryItem
	querySQL := `
		SELECT wl.id,
		       wl.library_name,
		       COALESCE(wl.library_meaning, '') AS library_meaning,
		       wl.status,
		       wl.word_count,
		       wl.created_by,
		       wl.create_time,
		       wl.update_time
		FROM word_library wl
		WHERE 1=1` + whereSQL + `
		ORDER BY wl.id ASC
		LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, (page-1)*pageSize)
	if err := db.Raw(querySQL, queryArgs...).Scan(&items).Error; err != nil {
		response.FailWithMessage("获取词库列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func (a *WordLibraryApi) Words(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	libraryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || libraryID <= 0 {
		response.FailWithMessage("词库 ID 不正确", c)
		return
	}

	page, pageSize := parseClozePagination(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	orderSQL := wordLibraryWordOrder(c.Query("sortBy"), c.Query("sortOrder"))
	whereSQL, args := wordLibraryWordWhere(libraryID, keyword)
	bestSentenceJoinSQL := wordLibraryBestSentenceJoinSQL()

	if err := ensureWordCleanBestSentenceTable(db); err != nil {
		response.FailWithMessage("初始化最佳例句表失败: "+err.Error(), c)
		return
	}

	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM word w"+bestSentenceJoinSQL+" WHERE 1=1"+whereSQL, args...).Scan(&total).Error; err != nil {
		response.FailWithMessage("获取单词总数失败: "+err.Error(), c)
		return
	}

	var items []WordLibraryWordItem
	querySQL := `
		SELECT w.id,
		       w.library_id,
		       w.word,
		       w.meaning,
		       COALESCE(w.pronunciation_us, '') AS pronunciation_us,
		       COALESCE(w.pronunciation_uk, '') AS pronunciation_uk,
		       w.frequency,
		       w.difficulty,
		       w.status,
		       COALESCE(w.phrase, '') AS phrase,
		       COALESCE(w.phrase_translation, '') AS phrase_translation,
		       ` + wordLibraryBestSentenceSelectSQL() + `,
		       w.create_time,
		       w.update_time
		FROM word w` + bestSentenceJoinSQL + `
		WHERE 1=1` + whereSQL + `
		ORDER BY ` + orderSQL + `
		LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, (page-1)*pageSize)
	if err := db.Raw(querySQL, queryArgs...).Scan(&items).Error; err != nil {
		response.FailWithMessage("获取单词列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func (a *WordLibraryApi) CleanWords(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	page, pageSize := parseClozePagination(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	pepDifficulty, _ := strconv.Atoi(c.Query("pepDifficulty"))
	sourceGroup := strings.TrimSpace(c.Query("sourceGroup"))
	difficultyMin, _ := strconv.Atoi(c.Query("difficultyMin"))
	difficultyMax, _ := strconv.Atoi(c.Query("difficultyMax"))
	orderSQL := wordCleanOrder(c.Query("sortBy"), c.Query("sortOrder"))
	whereSQL, args := wordCleanWhere(keyword, pepDifficulty, sourceGroup, difficultyMin, difficultyMax)

	if err := ensureWordCleanBestSentenceTable(db); err != nil {
		response.FailWithMessage("初始化最佳例句表失败: "+err.Error(), c)
		return
	}

	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM word_clean wc WHERE 1=1"+whereSQL, args...).Scan(&total).Error; err != nil {
		response.FailWithMessage("获取去重单词总数失败: "+err.Error(), c)
		return
	}

	var items []WordCleanItem
	querySQL := `
		SELECT wc.id,
		       wc.word,
		       wc.meaning,
		       wc.difficulty,
		       wc.frequency,
		       COALESCE(wc.sentence, '') AS sentence,
		       wc.pep_difficulty,
		       COALESCE(wc.pep_difficulty_label, '') AS pep_difficulty_label,
		       wc.source_difficulty,
		       COALESCE(wc.source_difficulty_label, '') AS source_label,
		       wcbs.id AS best_sentence_id,
		       wcbs.source_sentence_id AS best_source_sentence_id,
		       COALESCE(wcbs.source_model_name, '') AS best_source_model_name,
		       COALESCE(wcbs.sentence, '') AS best_sentence,
		       COALESCE(wcbs.sentence_translation, '') AS best_sentence_translation,
		       wcbs.score AS best_sentence_score,
		       COALESCE(wcbs.score_reason, '') AS best_sentence_score_reason,
		       COALESCE(wcbs.score_model_name, '') AS best_sentence_score_model_name,
		       wcbs.scored_at AS best_sentence_scored_at,
		       COALESCE(wcbs.tts_status, '') AS best_sentence_tts_status,
		       COALESCE(wcbs.tts_bucket, '') AS best_sentence_tts_bucket,
		       COALESCE(wcbs.tts_object_key, '') AS best_sentence_tts_object_key,
		       COALESCE(wcbs.tts_object_url, '') AS best_sentence_tts_object_url,
		       COALESCE(wcbs.tts_content_type, '') AS best_sentence_tts_content_type,
		       wcbs.tts_file_size AS best_sentence_tts_file_size,
		       wcbs.tts_duration_ms AS best_sentence_tts_duration_ms,
		       wcbs.tts_generated_at AS best_sentence_tts_generated_at,
		       COALESCE(wcbs.tts_error_message, '') AS best_sentence_tts_error_message` + wordCleanTTSSelectSQL() + `
		FROM word_clean wc
		LEFT JOIN word_clean_best_sentence wcbs ON wcbs.word_clean_id = wc.id` + wordCleanTTSJoinSQL() + `
		WHERE 1=1` + whereSQL + `
		ORDER BY ` + orderSQL + `
		LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, (page-1)*pageSize)
	if err := db.Raw(querySQL, queryArgs...).Scan(&items).Error; err != nil {
		response.FailWithMessage("获取去重单词列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func wordCleanTTSSelectSQL() string {
	return `,
	       COALESCE(wct.status, '') AS word_tts_status,
	       COALESCE(wct.tts_bucket, '') AS word_tts_bucket,
	       COALESCE(wct.tts_object_key, '') AS word_tts_object_key,
	       COALESCE(wct.tts_object_url, '') AS word_tts_object_url`
}

func wordCleanTTSJoinSQL() string {
	return " LEFT JOIN word_clean_tts wct ON wct.word_clean_id = wc.id"
}

func (a *WordLibraryApi) CleanWordSentences(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	wordCleanID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || wordCleanID <= 0 {
		response.FailWithMessage("去重单词 ID 不正确", c)
		return
	}

	items := make([]WordCleanSentenceItem, 0)
	if err := ensureWordCleanSentenceScoreColumns(db); err != nil {
		response.FailWithMessage("初始化评分字段失败: "+err.Error(), c)
		return
	}

	querySQL := `
		SELECT wcs.id,
		       wcs.word_clean_id,
		       wcs.word,
		       wcs.model_name,
		       wcs.sentence,
		       COALESCE(wcs.sentence_translation, '') AS sentence_translation,
		       wcs.score,
		       COALESCE(wcs.score_reason, '') AS score_reason,
		       COALESCE(wcs.score_model_name, '') AS score_model_name,
		       wcs.scored_at
		FROM word_clean_sentence wcs
		WHERE wcs.word_clean_id = ?
		ORDER BY wcs.id DESC`
	if err := db.Raw(querySQL, wordCleanID).Scan(&items).Error; err != nil {
		response.FailWithMessage("获取大模型造句结果失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(items, "获取成功", c)
}

func (a *WordLibraryApi) ScoreCleanSentences(c *gin.Context) {
	var req ScoreWordCleanSentencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("参数错误: "+err.Error(), c)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	agentResp, err := callWordAgentScoreCleanSentences(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(agentResp, "评分完成", c)
}

func callWordAgentScoreCleanSentences(req ScoreWordCleanSentencesRequest) (ScoreWordCleanSentencesResponse, error) {
	baseURL := global.GVA_CONFIG.WordAgent.ResolveBaseURL()
	requestBody, err := json.Marshal(req)
	if err != nil {
		return ScoreWordCleanSentencesResponse{}, fmt.Errorf("请求参数编码失败: %w", err)
	}

	client := http.Client{Timeout: global.GVA_CONFIG.WordAgent.Timeout()}
	httpReq, err := http.NewRequest(
		http.MethodPost,
		baseURL+"/v1/word-clean-sentences/score",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return ScoreWordCleanSentencesResponse{}, fmt.Errorf("创建 Python 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return ScoreWordCleanSentencesResponse{}, fmt.Errorf("调用 Python word-agent 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ScoreWordCleanSentencesResponse{}, fmt.Errorf("读取 Python 响应失败: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ScoreWordCleanSentencesResponse{}, fmt.Errorf("Python word-agent 返回失败: %s", extractWordAgentError(body))
	}

	var agentResp ScoreWordCleanSentencesResponse
	if err := json.Unmarshal(body, &agentResp); err != nil {
		return ScoreWordCleanSentencesResponse{}, fmt.Errorf("解析 Python 响应失败: %w", err)
	}
	return agentResp, nil
}

func ensureWordCleanSentenceScoreColumns(db *gorm.DB) error {
	statements := []string{
		"ALTER TABLE word_clean_sentence ADD COLUMN IF NOT EXISTS score integer NULL",
		"ALTER TABLE word_clean_sentence ADD COLUMN IF NOT EXISTS score_reason text NOT NULL DEFAULT ''",
		"ALTER TABLE word_clean_sentence ADD COLUMN IF NOT EXISTS score_model_name varchar(128) NOT NULL DEFAULT ''",
		"ALTER TABLE word_clean_sentence ADD COLUMN IF NOT EXISTS scored_at timestamptz NULL",
		"CREATE INDEX IF NOT EXISTS idx_word_clean_sentence_score ON word_clean_sentence(score)",
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureWordCleanBestSentenceTable(db *gorm.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS word_clean_best_sentence (
			id bigint GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
			word_clean_id bigint NOT NULL,
			word varchar(100) NOT NULL,
			meaning text NOT NULL DEFAULT '',
			source_sentence_id bigint NOT NULL,
			source_model_name varchar(160) NOT NULL,
			sentence text NOT NULL,
			sentence_translation text NOT NULL DEFAULT '',
			score integer NOT NULL,
			score_reason text NOT NULL DEFAULT '',
			score_model_name varchar(128) NOT NULL DEFAULT '',
			scored_at timestamptz NULL,
			tts_status varchar(32) NOT NULL DEFAULT 'pending',
			tts_provider varchar(64) NOT NULL DEFAULT '',
			tts_model varchar(128) NOT NULL DEFAULT '',
			tts_voice varchar(128) NOT NULL DEFAULT '',
			tts_audio_format varchar(32) NOT NULL DEFAULT '',
			tts_bucket varchar(128) NOT NULL DEFAULT '',
			tts_object_key text NOT NULL DEFAULT '',
			tts_object_url text NOT NULL DEFAULT '',
			tts_content_type varchar(128) NOT NULL DEFAULT '',
			tts_file_size bigint NULL,
			tts_duration_ms integer NULL,
			tts_generated_at timestamptz NULL,
			tts_error_message text NOT NULL DEFAULT '',
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		)`,
	}
	statements = append(statements, wordUniqueIndexStatements()...)
	statements = append(statements,
		"CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_best_sentence_word_clean ON word_clean_best_sentence(word_clean_id)",
		"CREATE INDEX IF NOT EXISTS idx_word_clean_best_sentence_source_sentence ON word_clean_best_sentence(source_sentence_id)",
		"CREATE INDEX IF NOT EXISTS idx_word_clean_best_sentence_score ON word_clean_best_sentence(score)",
		"CREATE INDEX IF NOT EXISTS idx_word_clean_best_sentence_tts_status ON word_clean_best_sentence(tts_status)",
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_word_clean_best_sentence_word_clean'
			) THEN
				ALTER TABLE word_clean_best_sentence
					ADD CONSTRAINT fk_word_clean_best_sentence_word_clean
					FOREIGN KEY (word_clean_id)
					REFERENCES word_clean(id)
					ON DELETE CASCADE;
			END IF;

			IF NOT EXISTS (
				SELECT 1
				FROM pg_constraint
				WHERE conname = 'fk_word_clean_best_sentence_source_sentence'
			) THEN
				ALTER TABLE word_clean_best_sentence
					ADD CONSTRAINT fk_word_clean_best_sentence_source_sentence
					FOREIGN KEY (source_sentence_id)
					REFERENCES word_clean_sentence(id)
					ON DELETE CASCADE;
			END IF;
		END $$`,
	)
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func wordLibraryWhere(keyword string) (string, []interface{}) {
	if keyword == "" {
		return "", nil
	}
	likeKeyword := "%" + keyword + "%"
	return " AND (wl.library_name ILIKE ? OR COALESCE(wl.library_meaning, '') ILIKE ? OR wl.id::text LIKE ?)", []interface{}{likeKeyword, likeKeyword, likeKeyword}
}

func wordLibraryWordWhere(libraryID int64, keyword string) (string, []interface{}) {
	args := []interface{}{libraryID}
	clauses := []string{"w.library_id = ?"}
	if keyword != "" {
		likeKeyword := "%" + keyword + "%"
		clauses = append(clauses, "(w.word ILIKE ? OR w.meaning ILIKE ? OR COALESCE(w.phrase, '') ILIKE ? OR COALESCE(wcbs.sentence, '') ILIKE ?)")
		args = append(args, likeKeyword, likeKeyword, likeKeyword, likeKeyword)
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

func wordLibraryBestSentenceJoinSQL() string {
	return " LEFT JOIN word_clean_best_sentence wcbs ON wcbs.word = w.word"
}

func wordLibraryBestSentenceSelectSQL() string {
	return `COALESCE(wcbs.sentence, '') AS sentence,
		       COALESCE(wcbs.sentence_translation, '') AS sentence_translation,
		       COALESCE(wcbs.tts_status, '') AS best_sentence_tts_status,
		       COALESCE(wcbs.tts_bucket, '') AS best_sentence_tts_bucket,
		       COALESCE(wcbs.tts_object_key, '') AS best_sentence_tts_object_key,
		       COALESCE(wcbs.tts_object_url, '') AS best_sentence_tts_object_url`
}

func wordUniqueIndexStatements() []string {
	return []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_word ON word_clean(word)",
		"CREATE UNIQUE INDEX IF NOT EXISTS uk_word_clean_best_sentence_word ON word_clean_best_sentence(word)",
	}
}

func wordCleanWhere(keyword string, pepDifficulty int, sourceGroup string, difficultyMin int, difficultyMax int) (string, []interface{}) {
	clauses := make([]string, 0, 5)
	args := make([]interface{}, 0, 10)
	if keyword != "" {
		likeKeyword := "%" + keyword + "%"
		clauses = append(clauses, "(wc.word ILIKE ? OR wc.meaning ILIKE ? OR COALESCE(wc.sentence, '') ILIKE ? OR COALESCE(wc.pep_difficulty_label, '') ILIKE ? OR COALESCE(wc.source_difficulty_label, '') ILIKE ?)")
		args = append(args, likeKeyword, likeKeyword, likeKeyword, likeKeyword, likeKeyword)
	}
	if pepDifficulty > 0 {
		clauses = append(clauses, "wc.pep_difficulty = ?")
		args = append(args, pepDifficulty)
	}
	if sourceDifficulty := wordCleanSourceDifficulty(sourceGroup); sourceDifficulty > 0 {
		clauses = append(clauses, "wc.source_difficulty = ?")
		args = append(args, sourceDifficulty)
	}
	if difficultyMin > 0 {
		clauses = append(clauses, "wc.difficulty >= ?")
		args = append(args, difficultyMin)
	}
	if difficultyMax > 0 {
		clauses = append(clauses, "wc.difficulty <= ?")
		args = append(args, difficultyMax)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

func wordCleanSourceDifficulty(sourceGroup string) int {
	switch strings.TrimSpace(sourceGroup) {
	case "cet4":
		return 25
	case "kaoyan":
		return 26
	case "bec":
		return 27
	case "cet6":
		return 28
	case "ielts":
		return 29
	case "tem4":
		return 30
	case "tem8":
		return 31
	case "toefl":
		return 32
	case "gmat":
		return 33
	case "sat":
		return 34
	case "gre":
		return 35
	case "other":
		return 36
	default:
		return 0
	}
}

func wordLibraryWordOrder(sortBy string, sortOrder string) string {
	var column string
	switch strings.TrimSpace(sortBy) {
	case "difficulty":
		column = "w.difficulty"
	case "frequency":
		column = "w.frequency"
	default:
		return "w.id ASC"
	}

	direction := "ASC"
	if strings.EqualFold(strings.TrimSpace(sortOrder), "desc") {
		direction = "DESC"
	}
	return column + " " + direction + ", w.id ASC"
}

func wordCleanOrder(sortBy string, sortOrder string) string {
	var column string
	switch strings.TrimSpace(sortBy) {
	case "difficulty":
		column = "wc.difficulty"
	case "frequency":
		column = "wc.frequency"
	case "pepDifficulty":
		column = "wc.pep_difficulty"
	case "sourceDifficulty":
		column = "wc.source_difficulty"
	default:
		return "wc.id ASC"
	}

	direction := "ASC"
	if strings.EqualFold(strings.TrimSpace(sortOrder), "desc") {
		direction = "DESC"
	}
	nullOrder := "NULLS LAST"
	if direction == "DESC" {
		nullOrder = "NULLS LAST"
	}
	return column + " " + direction + " " + nullOrder + ", wc.id ASC"
}
