package seed

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/conchi/study-workbench/internal/model"
	"github.com/conchi/study-workbench/internal/quiz"
)

// QuestionStats 是灌库结果，用来在 CLI 里核对覆盖情况。
type QuestionStats struct {
	Kps       int            // 可出题的知识点数
	Questions int            // 写入的题目数
	Skipped   int            // 生成器不支持、跳过的知识点数
	BySubject map[string]int // 各学科题目数
}

// Questions 为所有 quiz_enabled 学科的知识点生成题目并写入 questions 表。
// 幂等：按 (kp_id, code) upsert，重复执行只会覆盖内容、不会产生重复题。
func Questions(gdb *gorm.DB) (QuestionStats, error) {
	stats := QuestionStats{BySubject: map[string]int{}}

	type row struct {
		KpID        int64
		Title       string
		Payload     string
		Difficulty  int
		ModuleID    int64
		ModuleCode  string
		SubjectCode string
	}
	var rows []row
	err := gdb.Raw(`
		SELECT kp.id AS kp_id, kp.title, kp.payload, kp.difficulty,
		       m.id AS module_id, m.code AS module_code, s.code AS subject_code
		FROM knowledge_points kp
		JOIN modules m  ON m.id = kp.module_id
		JOIN subjects s ON s.id = m.subject_id
		WHERE s.quiz_enabled = ?
		ORDER BY s.order_no, m.order_no, kp.order_no`, true).Scan(&rows).Error
	if err != nil {
		return stats, err
	}

	// 干扰项来自同模块的兄弟节点，先按模块聚一遍标题。
	siblings := map[int64][]string{}
	for _, r := range rows {
		siblings[r.ModuleID] = append(siblings[r.ModuleID], r.Title)
	}

	err = gdb.Transaction(func(tx *gorm.DB) error {
		for _, r := range rows {
			specs := quiz.Generate(quiz.Kp{
				ID: r.KpID, Title: r.Title, Payload: r.Payload, Difficulty: r.Difficulty,
				SubjectCode: r.SubjectCode, ModuleCode: r.ModuleCode,
				Siblings: siblings[r.ModuleID],
			})
			if len(specs) == 0 {
				stats.Skipped++
				continue
			}
			stats.Kps++

			for _, sp := range specs {
				q := model.Question{
					KpID: r.KpID, Code: sp.Code, Type: "choice", Stem: sp.Stem,
					Options: sp.OptionsJSON(), Answer: sp.AnswerJSON(),
					Visual: sp.VisualJSON(), Speech: sp.SpeechJSON(),
					Difficulty: sp.Difficulty,
				}
				if err := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "kp_id"}, {Name: "code"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"type", "stem", "options", "answer", "visual", "speech", "difficulty",
					}),
				}).Create(&q).Error; err != nil {
					return err
				}
				stats.Questions++
				stats.BySubject[r.SubjectCode]++
			}
		}
		return nil
	})
	return stats, err
}
