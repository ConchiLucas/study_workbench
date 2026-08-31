package system

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/model/common/response"
	"github.com/conchi/go-react-template/server/utils/gormsafe"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ClozeResultApi struct{}

var (
	robEnglishWordDBMu sync.Mutex
	robEnglishWordDB   *gorm.DB
)

type ClozeResultUserSummary struct {
	UserID       int64     `json:"userId"`
	UserName     string    `json:"userName"`
	TotalCount   int64     `json:"totalCount"`
	LatestTime   time.Time `json:"latestTime"`
	LatestWords  []string  `json:"latestWords"`
	LatestSource string    `json:"latestSource"`
}

type ClozeResultItem struct {
	ID                    int64             `json:"id"`
	UserID                int64             `json:"userId"`
	UserName              string            `json:"userName"`
	Word                  string            `json:"word"`
	Words                 []string          `json:"words"`
	BlankWords            []string          `json:"blankWords"`
	Sentence              string            `json:"sentence"`
	TranslationZh         string            `json:"translationZh"`
	ExplanationZh         string            `json:"explanationZh"`
	ClozeSentence         string            `json:"clozeSentence"`
	ProviderID            string            `json:"providerId"`
	ProviderLabel         string            `json:"providerLabel"`
	Model                 string            `json:"model"`
	Source                string            `json:"source"`
	SourceEventIDs        []int64           `json:"sourceEventIds"`
	SourceAnswerDetailIDs []int64           `json:"sourceAnswerDetailIds"`
	SourceRecordIDs       []int64           `json:"sourceRecordIds"`
	SourceWordIDs         []int64           `json:"sourceWordIds"`
	SourceWords           []ClozeSourceWord `json:"sourceWords"`
	CreateTime            time.Time         `json:"createTime"`
	UpdateTime            time.Time         `json:"updateTime"`
}

type clozeResultUserRow struct {
	UserID          int64     `gorm:"column:user_id"`
	UserName        string    `gorm:"column:user_name"`
	TotalCount      int64     `gorm:"column:total_count"`
	LatestTime      time.Time `gorm:"column:latest_time"`
	LatestWordsJSON string    `gorm:"column:latest_words_json"`
	LatestSource    string    `gorm:"column:latest_source"`
}

type clozeResultItemRow struct {
	ID                        int64     `gorm:"column:id"`
	UserID                    int64     `gorm:"column:user_id"`
	UserName                  string    `gorm:"column:user_name"`
	Word                      string    `gorm:"column:word"`
	WordsJSON                 string    `gorm:"column:words_json"`
	BlankWordsJSON            string    `gorm:"column:blank_words_json"`
	Sentence                  string    `gorm:"column:sentence"`
	TranslationZh             string    `gorm:"column:translation_zh"`
	ExplanationZh             string    `gorm:"column:explanation_zh"`
	ClozeSentence             string    `gorm:"column:cloze_sentence"`
	ProviderID                string    `gorm:"column:provider_id"`
	ProviderLabel             string    `gorm:"column:provider_label"`
	Model                     string    `gorm:"column:model"`
	Source                    string    `gorm:"column:source"`
	SourceEventIDsJSON        string    `gorm:"column:source_event_ids_json"`
	SourceAnswerDetailIDsJSON string    `gorm:"column:source_answer_detail_ids_json"`
	SourceRecordIDsJSON       string    `gorm:"column:source_record_ids_json"`
	SourceWordIDsJSON         string    `gorm:"column:source_word_ids_json"`
	CreateTime                time.Time `gorm:"column:create_time"`
	UpdateTime                time.Time `gorm:"column:update_time"`
}

func (a *ClozeResultApi) Users(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	page, pageSize := parseClozePagination(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	whereSQL, args := clozeUserWhere(keyword)

	var total int64
	countSQL := "SELECT COUNT(*) FROM users u WHERE 1=1" + whereSQL
	if err := db.Raw(countSQL, args...).Scan(&total).Error; err != nil {
		response.FailWithMessage("获取用户总数失败: "+err.Error(), c)
		return
	}

	var rows []clozeResultUserRow
	querySQL := `
		WITH cloze_ranked AS (
			SELECT
				user_id,
				words_json,
				source,
				create_time,
				COUNT(*) OVER (PARTITION BY user_id) AS total_count,
				ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY create_time DESC, id DESC) AS row_no
			FROM sentence_cloze_item
			WHERE user_id IS NOT NULL
		),
		cloze_latest AS (
			SELECT user_id,
			       total_count,
			       create_time AS latest_time,
			       words_json AS latest_words_json,
			       source AS latest_source
			FROM cloze_ranked
			WHERE row_no = 1
		)
		SELECT u.id AS user_id,
		       COALESCE(NULLIF(u.nickname, ''), NULLIF(u.username, ''), CONCAT('用户 ', u.id::text)) AS user_name,
		       COALESCE(c.total_count, 0) AS total_count,
		       COALESCE(c.latest_time, u.create_time) AS latest_time,
		       COALESCE(c.latest_words_json, '[]') AS latest_words_json,
		       COALESCE(c.latest_source, '') AS latest_source
		FROM users u
		LEFT JOIN cloze_latest c ON c.user_id = u.id
		WHERE 1=1` + whereSQL + `
		ORDER BY c.latest_time DESC NULLS LAST, u.create_time DESC, u.id DESC
		LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, (page-1)*pageSize)
	if err := db.Raw(querySQL, queryArgs...).Scan(&rows).Error; err != nil {
		response.FailWithMessage("获取用户生成结果失败: "+err.Error(), c)
		return
	}

	items := make([]ClozeResultUserSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, ClozeResultUserSummary{
			UserID:       row.UserID,
			UserName:     row.UserName,
			TotalCount:   row.TotalCount,
			LatestTime:   row.LatestTime,
			LatestWords:  decodeStringJSONList(row.LatestWordsJSON),
			LatestSource: row.LatestSource,
		})
	}

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func (a *ClozeResultApi) Items(c *gin.Context) {
	db, err := openRobEnglishWordDB()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	page, pageSize := parseClozePagination(c)
	userID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("userId")), 10, 64)
	keyword := strings.TrimSpace(c.Query("keyword"))
	whereSQL, args := clozeItemWhere(userID, keyword)

	var total int64
	if err := db.Raw("SELECT COUNT(*) FROM sentence_cloze_item i WHERE 1=1"+whereSQL, args...).Scan(&total).Error; err != nil {
		response.FailWithMessage("获取结果总数失败: "+err.Error(), c)
		return
	}

	var rows []clozeResultItemRow
	querySQL := `
		SELECT i.id,
		       i.user_id,
		       COALESCE(NULLIF(u.nickname, ''), NULLIF(u.username, ''), NULLIF(i.user_name, ''), '') AS user_name,
		       i.word,
		       i.words_json,
		       i.blank_words_json,
		       i.sentence,
		       i.translation_zh,
		       COALESCE(i.explanation_zh, '') AS explanation_zh,
		       i.cloze_sentence,
		       COALESCE(i.provider_id, '') AS provider_id,
		       COALESCE(i.provider_label, '') AS provider_label,
		       COALESCE(i.model, '') AS model,
		       i.source,
		       i.source_event_ids_json,
		       i.source_answer_detail_ids_json,
		       i.source_record_ids_json,
		       i.source_word_ids_json,
		       i.create_time,
		       i.update_time
		FROM sentence_cloze_item i
		LEFT JOIN users u ON u.id = i.user_id
		WHERE 1=1` + whereSQL + `
		ORDER BY i.create_time DESC, i.id DESC
		LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, (page-1)*pageSize)
	if err := db.Raw(querySQL, queryArgs...).Scan(&rows).Error; err != nil {
		response.FailWithMessage("获取生成结果失败: "+err.Error(), c)
		return
	}

	items := make([]ClozeResultItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ClozeResultItem{
			ID:                    row.ID,
			UserID:                row.UserID,
			UserName:              row.UserName,
			Word:                  row.Word,
			Words:                 decodeStringJSONList(row.WordsJSON),
			BlankWords:            decodeStringJSONList(row.BlankWordsJSON),
			Sentence:              row.Sentence,
			TranslationZh:         row.TranslationZh,
			ExplanationZh:         row.ExplanationZh,
			ClozeSentence:         row.ClozeSentence,
			ProviderID:            row.ProviderID,
			ProviderLabel:         row.ProviderLabel,
			Model:                 row.Model,
			Source:                row.Source,
			SourceEventIDs:        decodeInt64JSONList(row.SourceEventIDsJSON),
			SourceAnswerDetailIDs: decodeInt64JSONList(row.SourceAnswerDetailIDsJSON),
			SourceRecordIDs:       decodeInt64JSONList(row.SourceRecordIDsJSON),
			SourceWordIDs:         decodeInt64JSONList(row.SourceWordIDsJSON),
			CreateTime:            row.CreateTime,
			UpdateTime:            row.UpdateTime,
		})
	}
	items = loadClozeSourceWords(global.GVA_DB, db, items)

	response.OkWithDetailed(response.PageResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, "获取成功", c)
}

func openRobEnglishWordDB() (*gorm.DB, error) {
	robEnglishWordDBMu.Lock()
	defer robEnglishWordDBMu.Unlock()

	if robEnglishWordDB != nil {
		return robEnglishWordDB, nil
	}

	pgsql := global.GVA_CONFIG.Pgsql
	if pgsql.Username == "" || pgsql.Path == "" || pgsql.Port == "" {
		return nil, fmt.Errorf("PostgreSQL 配置不完整")
	}
	db, err := gorm.Open(postgres.Open(pgsql.LinkDsn("rob_english_word")), gormsafe.Config(pgsql.GeneralDB))
	if err != nil {
		return nil, fmt.Errorf("连接 rob_english_word 数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("初始化 rob_english_word 连接池失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(time.Hour)

	robEnglishWordDB = db
	return robEnglishWordDB, nil
}

func clozeUserWhere(keyword string) (string, []interface{}) {
	if keyword == "" {
		return "", nil
	}
	likeKeyword := "%" + keyword + "%"
	return " AND (u.username ILIKE ? OR u.nickname ILIKE ? OR u.id::text LIKE ?)", []interface{}{likeKeyword, likeKeyword, likeKeyword}
}

func clozeItemWhere(userID int64, keyword string) (string, []interface{}) {
	var clauses []string
	var args []interface{}
	if userID > 0 {
		clauses = append(clauses, "i.user_id = ?")
		args = append(args, userID)
	}
	if keyword != "" {
		likeKeyword := "%" + keyword + "%"
		clauses = append(clauses, "(i.word ILIKE ? OR i.sentence ILIKE ? OR i.translation_zh ILIKE ? OR i.cloze_sentence ILIKE ?)")
		args = append(args, likeKeyword, likeKeyword, likeKeyword, likeKeyword)
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

func parseClozePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 500 {
		pageSize = 500
	}
	return page, pageSize
}

func decodeStringJSONList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}

func decodeInt64JSONList(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return []int64{}
	}
	var values []int64
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []int64{}
	}
	return values
}
