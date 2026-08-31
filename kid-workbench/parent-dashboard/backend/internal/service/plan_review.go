package service

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// ReviewQuestion 是家长端的题目视图，含正确答案。
// 和 QuestionDTO 分开定义而不是加开关：孩子端接口不该有任何路径能返回答案。
type ReviewQuestion struct {
	ID          int64           `json:"id"`
	Type        string          `json:"type"`
	Stem        string          `json:"stem"`
	Options     json.RawMessage `json:"options"`
	Visual      json.RawMessage `json:"visual"`
	AnswerIndex int             `json:"answer_index"`
}

type PlanReviewItem struct {
	Seq         int            `json:"seq"`
	Status      string         `json:"status"` // pending|correct|wrong
	Bucket      string         `json:"bucket"` // review|shaky|learning|new
	Tries       int            `json:"tries"`
	CostMs      int            `json:"cost_ms"`
	Picks       []int          `json:"picks"` // 按作答顺序；旧数据为空
	AnsweredAt  *time.Time     `json:"answered_at"`
	KpID        int64          `json:"kp_id"`
	KpTitle     string         `json:"kp_title"`
	SubjectCode string         `json:"subject_code"`
	SubjectName string         `json:"subject_name"`
	Question    ReviewQuestion `json:"question"`
}

type PlanReview struct {
	Plan  PlanDTO           `json:"plan"`
	Items []PlanReviewItem  `json:"items"`
}

// Review 给家长端复盘：含正确答案和孩子选过的选项。
func (s *PlanService) Review(childID, planID int64) (PlanReview, error) {
	p, err := s.loadPlan(s.repo.DB(), childID, planID)
	if err != nil {
		return PlanReview{}, err
	}

	out := PlanReview{Plan: toPlanDTO(p), Items: []PlanReviewItem{}}

	type row struct {
		Seq         int
		Status      string
		Bucket      string
		Tries       int
		CostMs      int
		Picks       string
		AnsweredAt  *time.Time
		KpID        int64
		KpTitle     string
		SubjectCode string
		SubjectName string
		QuestionID  int64
		Type        string
		Stem        string
		Options     string
		Visual      string
		Answer      string
	}
	var rows []row
	if err := s.repo.DB().Raw(`
		SELECT pi.seq, pi.status, pi.bucket, pi.tries, pi.cost_ms, pi.picks, pi.answered_at,
		       pi.kp_id, kp.title AS kp_title,
		       s.code AS subject_code, s.name AS subject_name,
		       q.id AS question_id, q.type, q.stem, q.options, q.visual, q.answer
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
		answerIndex, err := parseAnswerIndex(r.Answer)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, PlanReviewItem{
			Seq: r.Seq, Status: r.Status, Bucket: r.Bucket, Tries: r.Tries,
			CostMs: r.CostMs, Picks: parsePicks(r.Picks), AnsweredAt: r.AnsweredAt,
			KpID: r.KpID, KpTitle: r.KpTitle,
			SubjectCode: r.SubjectCode, SubjectName: r.SubjectName,
			Question: ReviewQuestion{
				ID: r.QuestionID, Type: r.Type, Stem: r.Stem,
				Options: rawJSON(r.Options, "[]"), Visual: rawJSON(r.Visual, "{}"),
				AnswerIndex: answerIndex,
			},
		})
	}
	return out, nil
}

func parsePicks(s string) []int {
	if s == "" {
		return []int{}
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}
