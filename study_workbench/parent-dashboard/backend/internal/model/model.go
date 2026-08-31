package model

import "time"

type User struct {
	ID           int64 `gorm:"primaryKey"`
	Phone        string
	Nickname     string
	PasswordHash string
	CreatedAt    time.Time
}

func (User) TableName() string { return "users" }

type Child struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	Name      string     `json:"name"`
	Grade     string     `json:"grade"`
	AvatarURL string     `gorm:"column:avatar_url" json:"avatar_url"`
	Birthday  *time.Time `json:"birthday"`
	Flowers   int        `json:"flowers"`
	CreatedAt time.Time  `json:"created_at"`
}

func (Child) TableName() string { return "children" }

type Subject struct {
	ID      int64 `gorm:"primaryKey"`
	Code    string
	Name    string
	Icon    string
	OrderNo int
	// QuizEnabled 标记该学科能否自动出题。科普/古诗这类题面依赖真实内容库，
	// 暂时进不了答题计划，但在家长看板照常显示。
	QuizEnabled bool
}

func (Subject) TableName() string { return "subjects" }

type Module struct {
	ID        int64 `gorm:"primaryKey"`
	SubjectID int64
	Code      string
	Name      string
	OrderNo   int
}

func (Module) TableName() string { return "modules" }

type KnowledgePoint struct {
	ID         int64 `gorm:"primaryKey"`
	ModuleID   int64
	Code       string
	Title      string
	Payload    string
	Difficulty int
	OrderNo    int
}

func (KnowledgePoint) TableName() string { return "knowledge_points" }

type Question struct {
	ID   int64 `gorm:"primaryKey"`
	KpID int64 `gorm:"column:kp_id"`
	// Code 是同一知识点下的变体标识（calc/story/listen…），与 KpID 一起唯一，
	// 让重复灌库变成幂等的 upsert。
	Code       string
	Type       string
	Stem       string
	Options    string
	Answer     string
	Visual     string
	Speech     string
	MediaURL   string `gorm:"column:media_url"`
	Difficulty int
}

func (Question) TableName() string { return "questions" }

type Attempt struct {
	ID         int64 `gorm:"primaryKey"`
	ChildID    int64
	KpID       int64 `gorm:"column:kp_id"`
	QuestionID *int64
	IsCorrect  bool
	CostMs     int
	Source     string
	ClientID   string `gorm:"column:client_id"`
	CreatedAt  time.Time
}

func (Attempt) TableName() string { return "attempts" }

type MasteryState struct {
	ChildID      int64 `gorm:"primaryKey"`
	KpID         int64 `gorm:"primaryKey;column:kp_id"`
	Status       string
	Attempts     int
	Correct      int
	Streak       int
	BestStreak   int
	Ease         float64
	IntervalDays int
	DueAt        *time.Time
	FirstSeenAt  *time.Time
	MasteredAt   *time.Time
	UpdatedAt    time.Time
}

func (MasteryState) TableName() string { return "mastery_states" }

// MasterySkill 识字等学科按题型拆分的掌握度（字 × 技能）。
type MasterySkill struct {
	ChildID      int64  `gorm:"primaryKey"`
	KpID         int64  `gorm:"primaryKey;column:kp_id"`
	SkillCode    string `gorm:"primaryKey;column:skill_code"`
	Status       string
	Attempts     int
	Correct      int
	Streak       int
	BestStreak   int
	Ease         float64
	IntervalDays int
	DueAt        *time.Time
	FirstSeenAt  *time.Time
	MasteredAt   *time.Time
	UpdatedAt    time.Time
}

func (MasterySkill) TableName() string { return "mastery_skills" }

type DailyStat struct {
	ChildID       int64     `gorm:"primaryKey"`
	StatDate      time.Time `gorm:"primaryKey;type:date"`
	PracticeSec   int
	Attempts      int
	Correct       int
	NewlyMastered int
	ReviewDone    int
	CheckedIn     bool
}

func (DailyStat) TableName() string { return "daily_stats" }

type DailyTask struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	ChildID       int64     `json:"child_id"`
	TaskDate      time.Time `gorm:"type:date" json:"task_date"`
	Title         string    `json:"title"`
	SubjectID     *int64    `json:"subject_id"`
	TargetCount   int       `json:"target_count"`
	DoneCount     int       `json:"done_count"`
	RewardFlowers int       `json:"reward_flowers"`
	Status        string    `json:"status"`
}

func (DailyTask) TableName() string { return "daily_tasks" }

// StudyPlan 是某个孩子某一天的一份答题计划。
// SeqNo 为 1 是当天的主计划，2 及以上是家长手动加的"加餐"。
type StudyPlan struct {
	ID      int64 `gorm:"primaryKey"`
	ChildID int64
	// PlanDate 用字符串存 YYYY-MM-DD。两个方言对 DATE 的读写行为不一致，
	// 统一走文本避免时区把日期带偏一天，与 daily_stats 的处理方式一致。
	PlanDate     string `gorm:"type:date"`
	SeqNo        int
	Status       string
	TargetCount  int
	DoneCount    int
	CorrectCount int
	Stars        int
	DurationSec  int
	CreatedAt    time.Time
	StartedAt    *time.Time
	CompletedAt  *time.Time
}

func (StudyPlan) TableName() string { return "study_plans" }

// PlanItem 是计划里的一道题。生成时就把 question_id 固定下来，
// 这样中途刷新页面、换设备继续，看到的都是同一份题。
type PlanItem struct {
	ID         int64 `gorm:"primaryKey"`
	PlanID     int64
	Seq        int
	KpID       int64 `gorm:"column:kp_id"`
	QuestionID int64
	Bucket     string
	Status     string
	Tries      int
	CostMs     int
	// Picks 是孩子选过的选项下标，逗号分隔，如 "3,1"。家长端复盘用。
	Picks      string
	AnsweredAt *time.Time
}

func (PlanItem) TableName() string { return "plan_items" }

type Reward struct {
	ID      int64  `gorm:"primaryKey" json:"id"`
	ChildID int64  `json:"child_id"`
	Name    string `json:"name"`
	Cost    int    `json:"cost"`
	Stock   int    `json:"stock"`
}

func (Reward) TableName() string { return "rewards" }

type FlowerLedger struct {
	ID        int64 `gorm:"primaryKey"`
	ChildID   int64
	Delta     int
	Reason    string
	RefType   string
	RefID     *int64
	CreatedAt time.Time
}

func (FlowerLedger) TableName() string { return "flower_ledger" }
