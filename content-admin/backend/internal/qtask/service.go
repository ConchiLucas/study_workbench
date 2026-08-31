package qtask

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"
)

const TargetCount = 10

const StatusDraft = "draft"
const StatusPublished = "published"

type CreateInput struct {
	SubjectCode string
	ModuleCode  string
	Title       string
}

type TaskDTO struct {
	ID          int64     `json:"id"`
	SubjectCode string    `json:"subjectCode"`
	Title       string    `json:"title"`
	ModuleCode  string    `json:"moduleCode"`
	ModuleName  string    `json:"moduleName"`
	TargetCount int       `json:"targetCount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Items       []ItemDTO `json:"items,omitempty"`
}

type ItemDTO struct {
	Seq         int             `json:"seq"`
	KpID        int64           `json:"kpId"`
	QuestionID  int64           `json:"questionId"`
	CharText    string          `json:"charText"`
	Code        string          `json:"code"`
	Stem        string          `json:"stem"`
	Options     json.RawMessage `json:"options"`
	AnswerIndex int             `json:"answerIndex"`
	Speech      json.RawMessage `json:"speech"`
}

type ModuleOption struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Order int    `json:"order"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type taskRow struct {
	ID          int64
	SubjectCode string
	Title       string
	ModuleCode  string
	ModuleName  string
	TargetCount int
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type candidateRow struct {
	QuestionID int64
	KpID       int64
	CharText   string
	Code       string
	Stem       string
	Options    string
	Answer     string
	Speech     string
}

type itemRow struct {
	Seq        int
	KpID       int64
	QuestionID int64
	CharText   string
	Code       string
	Stem       string
	Options    string
	Answer     string
	Speech     string
}

type moduleRow struct {
	Code       string
	Name       string
	ModuleOrder int `gorm:"column:module_order"`
}

func (s *Service) Create(in CreateInput) (TaskDTO, error) {
	if in.SubjectCode != "literacy" {
		return TaskDTO{}, fmt.Errorf("暂仅支持 literacy 学科")
	}
	moduleCode := strings.TrimSpace(in.ModuleCode)
	if moduleCode == "" {
		return TaskDTO{}, fmt.Errorf("moduleCode 不能为空")
	}

	var mod moduleRow
	err := s.db.Raw(`
		SELECT m.code AS code, m.name AS name, m.order_no AS module_order
		FROM modules m
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = ? AND m.code = ?
	`, in.SubjectCode, moduleCode).Scan(&mod).Error
	if err != nil {
		return TaskDTO{}, err
	}
	if mod.Code == "" {
		return TaskDTO{}, gorm.ErrRecordNotFound
	}

	candidates, err := s.fetchCandidates(in.SubjectCode, moduleCode)
	if err != nil {
		return TaskDTO{}, err
	}
	if len(candidates) < TargetCount {
		return TaskDTO{}, fmt.Errorf("本组可出题仅 %d 道，需要 %d 道", len(candidates), TargetCount)
	}

	picked := pickCandidates(candidates, TargetCount)

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = fmt.Sprintf("识字 · %s", mod.Name)
	}

	now := time.Now().UTC()
	var taskID int64
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Raw(`
				INSERT INTO question_tasks (subject_code, title, module_code, module_name, target_count, status, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				RETURNING id
			`, in.SubjectCode, title, moduleCode, mod.Name, TargetCount, StatusDraft, now, now).Scan(&taskID).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Exec(`
				INSERT INTO question_tasks (subject_code, title, module_code, module_name, target_count, status, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, in.SubjectCode, title, moduleCode, mod.Name, TargetCount, StatusDraft, now, now).Error; err != nil {
				return err
			}
			if err := tx.Raw(`SELECT last_insert_rowid()`).Scan(&taskID).Error; err != nil {
				return err
			}
		}
		for i, c := range picked {
			if err := tx.Exec(`
				INSERT INTO question_task_items (task_id, seq, kp_id, question_id)
				VALUES (?, ?, ?, ?)
			`, taskID, i+1, c.KpID, c.QuestionID).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return TaskDTO{}, err
	}
	return s.Get(taskID)
}

func (s *Service) List(subject, status string) ([]TaskDTO, error) {
	q := s.db.Table("question_tasks").Select(`
		id, subject_code, title, module_code, module_name, target_count, status, created_at, updated_at
	`)
	if subject = strings.TrimSpace(subject); subject != "" {
		q = q.Where("subject_code = ?", subject)
	}
	if status = strings.TrimSpace(status); status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []taskRow
	if err := q.Order("updated_at DESC, id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]TaskDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, taskToDTO(r, nil))
	}
	return out, nil
}

func (s *Service) Get(id int64) (TaskDTO, error) {
	var row taskRow
	err := s.db.Raw(`
		SELECT id, subject_code, title, module_code, module_name, target_count, status, created_at, updated_at
		FROM question_tasks WHERE id = ?
	`, id).Scan(&row).Error
	if err != nil {
		return TaskDTO{}, err
	}
	if row.ID == 0 {
		return TaskDTO{}, gorm.ErrRecordNotFound
	}

	var items []itemRow
	err = s.db.Raw(`
		SELECT i.seq, i.kp_id, i.question_id, kp.title AS char_text,
		       q.code, q.stem, q.options, q.answer, q.speech
		FROM question_task_items i
		JOIN questions q ON q.id = i.question_id
		JOIN knowledge_points kp ON kp.id = i.kp_id
		WHERE i.task_id = ?
		ORDER BY i.seq ASC
	`, id).Scan(&items).Error
	if err != nil {
		return TaskDTO{}, err
	}

	dtos := make([]ItemDTO, 0, len(items))
	for _, it := range items {
		dtos = append(dtos, itemToDTO(it))
	}
	return taskToDTO(row, dtos), nil
}

func (s *Service) Reshuffle(id int64) (TaskDTO, error) {
	task, err := s.loadTask(id)
	if err != nil {
		return TaskDTO{}, err
	}
	if task.Status != StatusDraft {
		return TaskDTO{}, fmt.Errorf("仅 draft 状态可重新抽题")
	}

	candidates, err := s.fetchCandidates(task.SubjectCode, task.ModuleCode)
	if err != nil {
		return TaskDTO{}, err
	}
	if len(candidates) < TargetCount {
		return TaskDTO{}, fmt.Errorf("本组可出题仅 %d 道，需要 %d 道", len(candidates), TargetCount)
	}
	picked := pickCandidates(candidates, TargetCount)

	now := time.Now().UTC()
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM question_task_items WHERE task_id = ?`, id).Error; err != nil {
			return err
		}
		for i, c := range picked {
			if err := tx.Exec(`
				INSERT INTO question_task_items (task_id, seq, kp_id, question_id)
				VALUES (?, ?, ?, ?)
			`, id, i+1, c.KpID, c.QuestionID).Error; err != nil {
				return err
			}
		}
		return tx.Exec(`UPDATE question_tasks SET updated_at = ? WHERE id = ?`, now, id).Error
	})
	if err != nil {
		return TaskDTO{}, err
	}
	return s.Get(id)
}

func (s *Service) Publish(id int64) (TaskDTO, error) {
	task, err := s.loadTask(id)
	if err != nil {
		return TaskDTO{}, err
	}
	if task.Status == StatusPublished {
		return TaskDTO{}, fmt.Errorf("任务已是 published 状态")
	}
	now := time.Now().UTC()
	if err := s.db.Exec(`UPDATE question_tasks SET status = ?, updated_at = ? WHERE id = ?`, StatusPublished, now, id).Error; err != nil {
		return TaskDTO{}, err
	}
	return s.Get(id)
}

func (s *Service) Unpublish(id int64) (TaskDTO, error) {
	task, err := s.loadTask(id)
	if err != nil {
		return TaskDTO{}, err
	}
	if task.Status != StatusPublished {
		return TaskDTO{}, fmt.Errorf("仅 published 状态可撤回")
	}
	now := time.Now().UTC()
	if err := s.db.Exec(`UPDATE question_tasks SET status = ?, updated_at = ? WHERE id = ?`, StatusDraft, now, id).Error; err != nil {
		return TaskDTO{}, err
	}
	return s.Get(id)
}

func (s *Service) Delete(id int64) error {
	task, err := s.loadTask(id)
	if err != nil {
		return err
	}
	if task.Status != StatusDraft {
		return fmt.Errorf("仅 draft 状态可删除")
	}
	res := s.db.Exec(`DELETE FROM question_tasks WHERE id = ?`, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Service) ListLiteracyModules() ([]ModuleOption, error) {
	var rows []moduleRow
	err := s.db.Raw(`
		SELECT m.code AS code, m.name AS name, m.order_no AS module_order
		FROM modules m
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = 'literacy'
		ORDER BY m.order_no ASC, m.id ASC
	`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]ModuleOption, 0, len(rows))
	for _, r := range rows {
		out = append(out, ModuleOption{Code: r.Code, Name: r.Name, Order: r.ModuleOrder})
	}
	return out, nil
}

func (s *Service) loadTask(id int64) (taskRow, error) {
	var row taskRow
	err := s.db.Raw(`
		SELECT id, subject_code, title, module_code, module_name, target_count, status, created_at, updated_at
		FROM question_tasks WHERE id = ?
	`, id).Scan(&row).Error
	if err != nil {
		return taskRow{}, err
	}
	if row.ID == 0 {
		return taskRow{}, gorm.ErrRecordNotFound
	}
	return row, nil
}

func (s *Service) fetchCandidates(subjectCode, moduleCode string) ([]candidateRow, error) {
	var rows []candidateRow
	err := s.db.Raw(`
		SELECT q.id AS question_id, q.kp_id, kp.title AS char_text,
		       q.code, q.stem, q.options, q.answer, q.speech
		FROM questions q
		JOIN knowledge_points kp ON kp.id = q.kp_id
		JOIN modules m ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.code = ? AND m.code = ?
		  AND q.code IN ('glyph_sense', 'sense_char')
		ORDER BY q.id
	`, subjectCode, moduleCode).Scan(&rows).Error
	return rows, err
}

func pickCandidates(candidates []candidateRow, n int) []candidateRow {
	shuffled := make([]candidateRow, len(candidates))
	copy(shuffled, candidates)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled[:n]
}

func taskToDTO(r taskRow, items []ItemDTO) TaskDTO {
	return TaskDTO{
		ID:          r.ID,
		SubjectCode: r.SubjectCode,
		Title:       r.Title,
		ModuleCode:  r.ModuleCode,
		ModuleName:  r.ModuleName,
		TargetCount: r.TargetCount,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Items:       items,
	}
}

func itemToDTO(it itemRow) ItemDTO {
	opts := json.RawMessage("[]")
	if strings.TrimSpace(it.Options) != "" {
		opts = json.RawMessage(it.Options)
	}
	speech := json.RawMessage("{}")
	if strings.TrimSpace(it.Speech) != "" {
		speech = json.RawMessage(it.Speech)
	}
	return ItemDTO{
		Seq:         it.Seq,
		KpID:        it.KpID,
		QuestionID:  it.QuestionID,
		CharText:    it.CharText,
		Code:        it.Code,
		Stem:        it.Stem,
		Options:     opts,
		AnswerIndex: parseAnswerIndex(it.Answer),
		Speech:      speech,
	}
}

func parseAnswerIndex(raw string) int {
	var ans struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(raw), &ans); err != nil {
		return 0
	}
	return ans.Index
}
