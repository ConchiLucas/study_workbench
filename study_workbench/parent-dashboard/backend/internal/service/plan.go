package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/conchi/study-workbench/internal/mastery"
	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/plan"
	"github.com/conchi/study-workbench/internal/repo"
)

// 一天最多几份计划：主计划 + 2 次加餐。再多就不是"每天一点"而是刷题了。
const maxPlansPerDay = 3

// 没做完的任务只在这个窗口内还能在孩子端续做（今天、昨天、前天）。
const kidResumeWindowDays = 2

// 计划里的每道题最多答两次：错一次当场重来，第二次还错就给答案跳过。
// 追到第三次只会让孩子挫败。
const maxTriesPerItem = 2

var (
	ErrPlanNotFound      = errors.New("计划不存在")
	ErrPlanItemNotFound  = errors.New("题目不在该计划中")
	ErrNoCandidates      = errors.New("没有可出题的知识点，请先执行 seed -mode=questions")
	ErrTooManyPlansToday = fmt.Errorf("今天最多只能有 %d 份计划", maxPlansPerDay)
)

type PlanService struct {
	repo     *repo.Repo
	attempts *AttemptService
	quota    plan.Quota
	rules    plan.Rules
}

func NewPlanService(r *repo.Repo, a *AttemptService) *PlanService {
	return &PlanService{repo: r, attempts: a, quota: plan.DefaultQuota(), rules: plan.DefaultRules()}
}

type PlanDTO struct {
	ID           int64  `json:"id"`
	PlanDate     string `json:"plan_date"`
	SeqNo        int    `json:"seq_no"`
	Status       string `json:"status"`
	TargetCount  int    `json:"target_count"`
	DoneCount    int    `json:"done_count"`
	CorrectCount int    `json:"correct_count"`
	Stars        int    `json:"stars"`
	DurationSec  int    `json:"duration_sec"`
	// Flowers 是这份计划已经拿到的小红花总数，没做完是 0。
	// 由后端算好下发，避免前端再抄一份奖励规则、改了这边忘了那边。
	Flowers int `json:"flowers"`
}

type QuestionDTO struct {
	ID      int64           `json:"id"`
	Type    string          `json:"type"`
	Stem    string          `json:"stem"`
	Options json.RawMessage `json:"options"`
	Visual  json.RawMessage `json:"visual"`
	Speech  json.RawMessage `json:"speech"`
	// 刻意不含正确答案：判题在服务端做，孩子翻不到答案，掌握度数据才可信。
}

type ItemDTO struct {
	ID          int64       `json:"id"`
	Seq         int         `json:"seq"`
	Status      string      `json:"status"`
	Bucket      string      `json:"bucket"`
	Tries       int         `json:"tries"`
	KpID        int64       `json:"kp_id"`
	KpTitle     string      `json:"kp_title"`
	SubjectCode string      `json:"subject_code"`
	SubjectName string      `json:"subject_name"`
	Question    QuestionDTO `json:"question"`
}

type PlanDetail struct {
	Plan  PlanDTO   `json:"plan"`
	Items []ItemDTO `json:"items"`
}

// Today 返回当天"当前那份"计划：已有计划时给序号最大的那份，一份都没有就现场生成主计划。
//
// 取最大序号而不是固定取主计划，是为了让加餐后刷新页面仍然停在加餐上，
// 否则孩子刷一下就被扔回已经做完的主计划。
//
// 懒生成而不是定时任务：家里的 iPad 不一定每天开机，
// 定时任务只会造出一堆没人做的空计划，还会污染"完成率"统计。
func (s *PlanService) Today(childID int64) (PlanDetail, error) {
	date := today()
	var maxSeq int
	if err := s.repo.DB().Raw(
		`SELECT COALESCE(MAX(seq_no),0) FROM study_plans WHERE child_id = ? AND plan_date = ?`,
		childID, date).Scan(&maxSeq).Error; err != nil {
		return PlanDetail{}, err
	}
	if maxSeq == 0 {
		maxSeq = 1
	}
	return s.ensure(childID, date, maxSeq)
}

// Extra 加餐：在当天生成下一份计划。
func (s *PlanService) Extra(childID int64) (PlanDetail, error) {
	date := today()
	var maxSeq int
	if err := s.repo.DB().Raw(
		`SELECT COALESCE(MAX(seq_no),0) FROM study_plans WHERE child_id = ? AND plan_date = ?`,
		childID, date).Scan(&maxSeq).Error; err != nil {
		return PlanDetail{}, err
	}
	if maxSeq >= maxPlansPerDay {
		return PlanDetail{}, ErrTooManyPlansToday
	}
	return s.ensure(childID, date, maxSeq+1)
}

func (s *PlanService) ensure(childID int64, date string, seqNo int) (PlanDetail, error) {
	existing, err := s.findPlan(s.repo.DB(), childID, date, seqNo)
	if err == nil {
		return s.detail(existing)
	}
	if !errors.Is(err, ErrPlanNotFound) {
		return PlanDetail{}, err
	}

	created, err := s.generate(childID, date, seqNo)
	if err != nil {
		return PlanDetail{}, err
	}
	return s.detail(created)
}

func (s *PlanService) generate(childID int64, date string, seqNo int) (model.StudyPlan, error) {
	candidates, err := s.candidates(childID)
	if err != nil {
		return model.StudyPlan{}, err
	}
	if len(candidates) == 0 {
		return model.StudyPlan{}, ErrNoCandidates
	}

	// 轮换起始学科，让每天的学科组合不一样；加餐再往后错一位，
	// 免得加餐和主计划撞上同一批学科。
	rotation := dayOfYear(date) + seqNo
	chosen := plan.Compose(toPlanCandidates(candidates), s.quota, s.rules, rotation)
	if len(chosen) == 0 {
		return model.StudyPlan{}, ErrNoCandidates
	}

	byKp := map[int64]candidate{}
	kpIDs := make([]int64, 0, len(chosen))
	for _, c := range candidates {
		byKp[c.KpID] = c
	}
	for _, c := range chosen {
		kpIDs = append(kpIDs, c.KpID)
	}

	questions, err := s.questionsFor(kpIDs)
	if err != nil {
		return model.StudyPlan{}, err
	}

	row := model.StudyPlan{
		ChildID: childID, PlanDate: date, SeqNo: seqNo,
		Status: "pending", TargetCount: len(chosen), CreatedAt: time.Now(),
	}

	err = s.repo.Tx(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			// 两个设备同时打开答题页会撞唯一键，此时直接用已有的那份。
			if isUniqueViolation(err) {
				got, ferr := s.findPlan(tx, childID, date, seqNo)
				if ferr != nil {
					return ferr
				}
				row = got
				return nil
			}
			return err
		}

		seq := 0
		for _, c := range chosen {
			variants := questions[c.KpID]
			if len(variants) == 0 {
				continue
			}
			// 换着变体出：同一个知识点第二次遇到时不该是完全相同的题，
			// 否则孩子会记住"选第几个"而不是记住知识点。
			pick := variants[(byKp[c.KpID].Attempts+rotation)%len(variants)]
			seq++
			if err := tx.Create(&model.PlanItem{
				PlanID: row.ID, Seq: seq, KpID: c.KpID,
				QuestionID: pick, Bucket: c.Bucket, Status: "pending",
			}).Error; err != nil {
				return err
			}
		}
		if seq != row.TargetCount {
			row.TargetCount = seq
			if err := tx.Model(&row).UpdateColumn("target_count", seq).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return row, err
}

// candidate 是一个可进计划的知识点。
type candidate struct {
	KpID        int64
	SubjectCode string
	SubjectOrd  int
	ModuleOrd   int
	KpOrd       int
	Status      string
	Accuracy    float64
	Attempts    int
	DueAt       *time.Time
}

func (c candidate) bucket() string {
	switch c.Status {
	case "review_due":
		return plan.BucketReview
	case "shaky":
		return plan.BucketShaky
	case "learning":
		return plan.BucketLearning
	case "not_started":
		return plan.BucketNew
	}
	return ""
}

func toPlanCandidates(in []candidate) []plan.Candidate {
	out := make([]plan.Candidate, 0, len(in))
	for _, c := range in {
		out = append(out, plan.Candidate{
			KpID: c.KpID, SubjectCode: c.SubjectCode, Bucket: c.bucket(),
		})
	}
	return out
}

// candidates 取出所有能进计划的知识点，并按"该练的程度"排好序。
//
// 排序在 Go 里做而不是写进 SQL：四个桶的优先级依据不同（复习看到期时间、
// 易错看正确率、新知看教学顺序），塞进一条 ORDER BY 既难读又踩两个方言
// 对 NULL 排序的差异。候选最多几百条，内存里排完全够。
func (s *PlanService) candidates(childID int64) ([]candidate, error) {
	var rows []candidate
	err := s.repo.DB().Raw(`
		SELECT kp.id AS kp_id, s.code AS subject_code,
		       s.order_no AS subject_ord, m.order_no AS module_ord, kp.order_no AS kp_ord,
		       `+effectiveStatusSQL+` AS status,
		       CASE WHEN COALESCE(ms.attempts,0) > 0
		            THEN CAST(ms.correct AS REAL) / ms.attempts ELSE 0 END AS accuracy,
		       COALESCE(ms.attempts,0) AS attempts,
		       ms.due_at
		FROM knowledge_points kp
		JOIN modules m  ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		LEFT JOIN mastery_states ms ON ms.kp_id = kp.id AND ms.child_id = ?
		WHERE s.quiz_enabled = ?
		  AND EXISTS (SELECT 1 FROM questions q WHERE q.kp_id = kp.id)`,
		childID, true).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]candidate, 0, len(rows))
	for _, r := range rows {
		// 已掌握且没到复习点的知识点不用练，留出时间给该练的。
		if r.bucket() == "" {
			continue
		}
		out = append(out, r)
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Status != b.Status {
			return bucketRank(a.bucket()) < bucketRank(b.bucket())
		}
		switch a.bucket() {
		case plan.BucketReview:
			// 越早到期的越该先复习。
			return dueBefore(a.DueAt, b.DueAt)
		case plan.BucketShaky, plan.BucketLearning:
			// 正确率越低越该练；同正确率时错得多的优先。
			if a.Accuracy != b.Accuracy {
				return a.Accuracy < b.Accuracy
			}
			return a.Attempts > b.Attempts
		default:
			// 新知识点按教学顺序推进，不跳着学。
			if a.SubjectOrd != b.SubjectOrd {
				return a.SubjectOrd < b.SubjectOrd
			}
			if a.ModuleOrd != b.ModuleOrd {
				return a.ModuleOrd < b.ModuleOrd
			}
			return a.KpOrd < b.KpOrd
		}
	})
	return out, nil
}

func bucketRank(b string) int {
	switch b {
	case plan.BucketReview:
		return 0
	case plan.BucketShaky:
		return 1
	case plan.BucketLearning:
		return 2
	case plan.BucketNew:
		return 3
	}
	return 4
}

func dueBefore(a, b *time.Time) bool {
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return a.Before(*b)
}

// questionsFor 返回每个知识点的题目 ID 列表，按变体 code 排序保证顺序稳定。
func (s *PlanService) questionsFor(kpIDs []int64) (map[int64][]int64, error) {
	type row struct {
		KpID int64
		ID   int64
	}
	var rows []row
	if err := s.repo.DB().Raw(
		`SELECT kp_id, id FROM questions WHERE kp_id IN ? ORDER BY kp_id, code`, kpIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[int64][]int64{}
	for _, r := range rows {
		out[r.KpID] = append(out[r.KpID], r.ID)
	}
	return out, nil
}

func (s *PlanService) findPlan(tx *gorm.DB, childID int64, date string, seqNo int) (model.StudyPlan, error) {
	var row model.StudyPlan
	err := tx.Where("child_id = ? AND plan_date = ? AND seq_no = ?", childID, date, seqNo).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, ErrPlanNotFound
	}
	return row, err
}

func (s *PlanService) loadPlan(tx *gorm.DB, childID, planID int64) (model.StudyPlan, error) {
	var row model.StudyPlan
	err := tx.Where("id = ? AND child_id = ?", planID, childID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, ErrPlanNotFound
	}
	return row, err
}

func (s *PlanService) detail(p model.StudyPlan) (PlanDetail, error) {
	out := PlanDetail{Plan: toPlanDTO(p), Items: []ItemDTO{}}

	type row struct {
		ID          int64
		Seq         int
		Status      string
		Bucket      string
		Tries       int
		KpID        int64
		KpTitle     string
		SubjectCode string
		SubjectName string
		QuestionID  int64
		Type        string
		Stem        string
		Options     string
		Visual      string
		Speech      string
	}
	var rows []row
	if err := s.repo.DB().Raw(`
		SELECT pi.id, pi.seq, pi.status, pi.bucket, pi.tries,
		       pi.kp_id, kp.title AS kp_title,
		       s.code AS subject_code, s.name AS subject_name,
		       q.id AS question_id, q.type, q.stem, q.options, q.visual, q.speech
		FROM plan_items pi
		JOIN questions q         ON q.id = pi.question_id
		JOIN knowledge_points kp ON kp.id = pi.kp_id
		JOIN modules m           ON m.id = kp.module_id
		JOIN subjects s          ON s.id = m.subject_id
		WHERE pi.plan_id = ?
		ORDER BY pi.seq`, p.ID).Scan(&rows).Error; err != nil {
		return out, err
	}

	for _, r := range rows {
		out.Items = append(out.Items, ItemDTO{
			ID: r.ID, Seq: r.Seq, Status: r.Status, Bucket: r.Bucket, Tries: r.Tries,
			KpID: r.KpID, KpTitle: r.KpTitle,
			SubjectCode: r.SubjectCode, SubjectName: r.SubjectName,
			Question: QuestionDTO{
				ID: r.QuestionID, Type: r.Type, Stem: r.Stem,
				Options: rawJSON(r.Options, "[]"),
				Visual:  rawJSON(r.Visual, "{}"),
				Speech:  rawJSON(r.Speech, "{}"),
			},
		})
	}
	return out, nil
}

// Start 记录开始时间，用于统计真实用时。重复调用不会改写首次的时间。
func (s *PlanService) Start(childID, planID int64) (PlanDTO, error) {
	var out PlanDTO
	err := s.repo.Tx(func(tx *gorm.DB) error {
		p, err := s.loadPlan(tx, childID, planID)
		if err != nil {
			return err
		}
		if p.StartedAt == nil {
			now := time.Now()
			p.StartedAt = &now
			p.Status = "doing"
			if err := tx.Model(&model.StudyPlan{}).Where("id = ?", p.ID).
				Updates(map[string]any{"started_at": now, "status": "doing"}).Error; err != nil {
				return err
			}
		}
		out = toPlanDTO(p)
		return nil
	})
	return out, err
}

type AnswerInput struct {
	OptionIndex int `json:"option_index"`
	CostMs      int `json:"cost_ms"`
}

type AnswerResult struct {
	Correct     bool `json:"correct"`
	AnswerIndex int  `json:"answer_index"`
	// CanRetry 为真时前端提示"再试一次"，同一道题还能再答一遍。
	CanRetry bool     `json:"can_retry"`
	Tries    int      `json:"tries"`
	Status   string   `json:"status"`
	Mastery  StateDTO `json:"mastery"`
	Plan     PlanDTO  `json:"plan"`
}

// Answer 提交一道题。判题在服务端，并把这次作答写进 attempts——
// 掌握度、当日统计、家长看板都由那条链路自动更新，这里只维护计划自身的进度。
func (s *PlanService) Answer(childID, planID, itemID int64, in AnswerInput) (AnswerResult, error) {
	var out AnswerResult

	err := s.repo.Tx(func(tx *gorm.DB) error {
		p, err := s.loadPlan(tx, childID, planID)
		if err != nil {
			return err
		}

		var item model.PlanItem
		if err := tx.Where("id = ? AND plan_id = ?", itemID, planID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlanItemNotFound
			}
			return err
		}

		var q model.Question
		if err := tx.First(&q, item.QuestionID).Error; err != nil {
			return err
		}
		answerIndex, err := parseAnswerIndex(q.Answer)
		if err != nil {
			return err
		}
		out.AnswerIndex = answerIndex

		// 已经答完的题重复提交（网络重试、孩子连点）只回放结果，不重复计分。
		if item.Status != "pending" {
			out.Correct = item.Status == "correct"
			out.Tries, out.Status, out.CanRetry = item.Tries, item.Status, false
			out.Plan = toPlanDTO(p)
			return nil
		}

		correct := in.OptionIndex == answerIndex
		item.Tries++
		item.Picks = appendPick(item.Picks, in.OptionIndex)
		now := time.Now()

		dto, _, err := s.attempts.ApplyOne(tx, childID, AttemptInput{
			ClientID:   fmt.Sprintf("plan-%d-try%d", item.ID, item.Tries),
			KpID:       item.KpID,
			QuestionID: &q.ID,
			IsCorrect:  correct,
			CostMs:     in.CostMs,
			Source:     mastery.SourceQuiz,
			At:         now,
		})
		if err != nil {
			return err
		}
		out.Mastery = dto

		switch {
		case correct:
			item.Status = "correct"
		case item.Tries >= maxTriesPerItem:
			item.Status = "wrong"
		default:
			item.Status = "pending" // 还能再试一次
		}
		item.CostMs += in.CostMs
		if item.Status != "pending" {
			item.AnsweredAt = &now
		}

		if err := tx.Model(&model.PlanItem{}).Where("id = ?", item.ID).
			Updates(map[string]any{
				"tries": item.Tries, "status": item.Status, "picks": item.Picks,
				"cost_ms": item.CostMs, "answered_at": item.AnsweredAt,
			}).Error; err != nil {
			return err
		}

		// 只有题目真正结束（答对或用完两次机会）才算完成一题。
		if item.Status != "pending" {
			p.DoneCount++
			if item.Status == "correct" {
				p.CorrectCount++
			}
			if p.Status == "pending" {
				p.Status = "doing"
			}
			if err := tx.Model(&model.StudyPlan{}).Where("id = ?", p.ID).
				Updates(map[string]any{
					"done_count": p.DoneCount, "correct_count": p.CorrectCount,
					"status": p.Status,
				}).Error; err != nil {
				return err
			}
		}

		out.Correct = correct
		out.Tries = item.Tries
		out.Status = item.Status
		out.CanRetry = item.Status == "pending"
		out.Plan = toPlanDTO(p)
		return nil
	})
	return out, err
}

type FinishResult struct {
	Plan  PlanDTO `json:"plan"`
	Stars int     `json:"stars"`
	// Flowers 是这次结算"新发"的花，重复结算是 0。
	// 想显示这份计划总共拿了多少，用 Plan.Flowers。
	Flowers int `json:"flowers"`
}

// Finish 结算：算星星、记用时、发小红花。重复调用不会重复发花。
func (s *PlanService) Finish(childID, planID int64) (FinishResult, error) {
	var out FinishResult

	err := s.repo.Tx(func(tx *gorm.DB) error {
		p, err := s.loadPlan(tx, childID, planID)
		if err != nil {
			return err
		}
		if p.Status == "done" {
			out = FinishResult{Plan: toPlanDTO(p), Stars: p.Stars}
			return nil
		}

		now := time.Now()
		p.Stars = stars(p.CorrectCount, p.TargetCount)
		p.Status = "done"
		p.CompletedAt = &now
		p.DurationSec, err = s.timeOnTask(tx, p.ID)
		if err != nil {
			return err
		}

		if err := tx.Model(&model.StudyPlan{}).Where("id = ?", p.ID).
			Updates(map[string]any{
				"status": p.Status, "stars": p.Stars,
				"completed_at": now, "duration_sec": p.DurationSec,
			}).Error; err != nil {
			return err
		}

		flowers := flowersFor(p)
		planID := p.ID
		if err := s.repo.AddFlowers(tx, childID, flowers, "plan_done", "study_plan", &planID); err != nil {
			return err
		}

		out = FinishResult{Plan: toPlanDTO(p), Stars: p.Stars, Flowers: flowers}
		return nil
	})
	return out, err
}

// Todo 返回孩子现在还能做的任务：窗口内未完成的。已完成的不返回。
func (s *PlanService) Todo(childID int64) ([]PlanSummary, error) {
	to := today()
	from := time.Now().AddDate(0, 0, -kidResumeWindowDays).Format("2006-01-02")

	var rows []model.StudyPlan
	if err := s.repo.DB().
		Where("child_id = ? AND status IN ? AND plan_date >= ? AND plan_date <= ?",
			childID, []string{"pending", "doing"}, from, to).
		Order("plan_date DESC, seq_no").Find(&rows).Error; err != nil {
		return nil, err
	}
	return s.withSubjects(rows)
}

// Detail 按 id 取计划详情（孩子端用，不含正确答案）。
func (s *PlanService) Detail(childID, planID int64) (PlanDetail, error) {
	p, err := s.loadPlan(s.repo.DB(), childID, planID)
	if err != nil {
		return PlanDetail{}, err
	}
	return s.detail(p)
}

// History 给家长端的任务列表用。
func (s *PlanService) History(childID int64, from, to, status string) ([]PlanSummary, error) {
	if from == "" {
		from = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if to == "" {
		to = today()
	}

	q := s.repo.DB().
		Where("child_id = ? AND plan_date >= ? AND plan_date <= ?", childID, from, to)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var rows []model.StudyPlan
	if err := q.Order("plan_date DESC, seq_no").Find(&rows).Error; err != nil {
		return nil, err
	}

	return s.withSubjects(rows)
}

func (s *PlanService) withSubjects(rows []model.StudyPlan) ([]PlanSummary, error) {
	out := make([]PlanSummary, 0, len(rows))
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		out = append(out, PlanSummary{PlanDTO: toPlanDTO(r)})
		ids = append(ids, r.ID)
	}
	if len(ids) == 0 {
		return out, nil
	}

	subjects, err := s.subjectCounts(ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Subjects = subjects[out[i].ID]
		if out[i].Subjects == nil {
			out[i].Subjects = []SubjectCount{}
		}
	}
	return out, nil
}

type SubjectCount struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Count int    `json:"count"`
}

// PlanSummary 是任务列表行：计划摘要 + 学科分布。
// 不塞进 PlanDTO，避免孩子端 Answer 响应里白传 subjects。
type PlanSummary struct {
	PlanDTO
	Subjects []SubjectCount `json:"subjects"`
}

func (s *PlanService) subjectCounts(planIDs []int64) (map[int64][]SubjectCount, error) {
	type row struct {
		PlanID int64
		Code   string
		Name   string
		Icon   string
		Count  int
	}
	var rows []row
	if err := s.repo.DB().Raw(`
		SELECT pi.plan_id, s.code, s.name, s.icon, COUNT(1) AS count
		FROM plan_items pi
		JOIN knowledge_points kp ON kp.id = pi.kp_id
		JOIN modules m           ON m.id = kp.module_id
		JOIN subjects s          ON s.id = m.subject_id
		WHERE pi.plan_id IN ?
		GROUP BY pi.plan_id, s.code, s.name, s.icon, s.order_no
		ORDER BY pi.plan_id, s.order_no`, planIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[int64][]SubjectCount{}
	for _, r := range rows {
		out[r.PlanID] = append(out[r.PlanID], SubjectCount{
			Code: r.Code, Name: r.Name, Icon: r.Icon, Count: r.Count,
		})
	}
	return out, nil
}

func toPlanDTO(p model.StudyPlan) PlanDTO {
	return PlanDTO{
		ID: p.ID, PlanDate: normalizeDate(p.PlanDate), SeqNo: p.SeqNo, Status: p.Status,
		TargetCount: p.TargetCount, DoneCount: p.DoneCount, CorrectCount: p.CorrectCount,
		Stars: p.Stars, DurationSec: p.DurationSec, Flowers: flowersFor(p),
	}
}

// flowersFor：完成计划本身给 1 朵，星星越多多给——但一星也有奖励，
// 不让"今天做得不够好"变成"今天什么都没得到"。
func flowersFor(p model.StudyPlan) int {
	if p.Status != "done" {
		return 0
	}
	return 1 + p.Stars
}

// stars 三档：9~10 对三星，7~8 对两星，其余一星。
// 没有"零星"这档——这个年龄需要的是"今天也完成了"，不是打分。
func stars(correct, total int) int {
	if total <= 0 {
		return 1
	}
	switch ratio := float64(correct) / float64(total); {
	case ratio >= 0.9:
		return 3
	case ratio >= 0.7:
		return 2
	default:
		return 1
	}
}

// timeOnTask 用各题耗时之和，而不是开始到结束的墙上时间。
// 孩子做两题就跑去玩、晚上回来接着做是常态，墙上时间会算出"练了 12 小时"，
// 家长看板上的练习时长就没意义了。
func (s *PlanService) timeOnTask(tx *gorm.DB, planID int64) (int, error) {
	var ms int
	err := tx.Raw(`SELECT COALESCE(SUM(cost_ms),0) FROM plan_items WHERE plan_id = ?`, planID).
		Scan(&ms).Error
	return ms / 1000, err
}

func parseAnswerIndex(raw string) (int, error) {
	var a struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		return 0, fmt.Errorf("题目答案格式错误: %w", err)
	}
	return a.Index, nil
}

// appendPick 把本次选项下标追加到 picks CSV。
func appendPick(prev string, index int) string {
	s := fmt.Sprintf("%d", index)
	if prev == "" {
		return s
	}
	return prev + "," + s
}

func rawJSON(s, fallback string) json.RawMessage {
	if s == "" || !json.Valid([]byte(s)) {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(s)
}

func today() string { return time.Now().Format("2006-01-02") }

// normalizeDate 统一成 YYYY-MM-DD：Postgres 把 DATE 读成带时间的字符串，
// SQLite 读回的就是原样文本。
func normalizeDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func dayOfYear(date string) int {
	t, err := time.Parse("2006-01-02", normalizeDate(date))
	if err != nil {
		return 0
	}
	return t.YearDay()
}
