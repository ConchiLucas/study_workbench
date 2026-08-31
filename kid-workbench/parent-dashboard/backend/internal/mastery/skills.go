package mastery

// 识字两种题型技能。字的「完全掌握」要求二者都达到 mastered / review_due。
const (
	SkillGlyphSense = "glyph_sense" // 看字图选义图
	SkillSenseChar  = "sense_char"  // 看义图选字

	// SkillListenGlyph 已下线；历史作答仍可映射展示，但不计入掌握维度。
	SkillListenGlyph = "listen_glyph"
)

// 拼音两种题型技能（与 questions.code 一致）。
const (
	SkillPinyinInWord = "inword" // 听例字选音
	SkillPinyinListen = "listen" // 听单读选字母
)

// LiteracySkills 固定顺序，供矩阵两点展示与灌题使用。
var LiteracySkills = []string{
	SkillGlyphSense,
	SkillSenseChar,
}

// PinyinSkills 固定顺序。
var PinyinSkills = []string{
	SkillPinyinInWord,
	SkillPinyinListen,
}

// SkillsForSubject 返回该学科需要按技能记账的 code 列表；空表示按 KP 单行掌握。
func SkillsForSubject(subjectCode string) []string {
	switch subjectCode {
	case "literacy":
		return append([]string{}, LiteracySkills...)
	case "pinyin":
		return append([]string{}, PinyinSkills...)
	default:
		return nil
	}
}

// SkillFromQuestionCode 把题目 code 映射到技能。
func SkillFromQuestionCode(code string) string {
	switch code {
	case SkillGlyphSense:
		return SkillGlyphSense
	case SkillSenseChar:
		return SkillSenseChar
	case SkillPinyinInWord:
		return SkillPinyinInWord
	case SkillPinyinListen:
		return SkillPinyinListen
	case SkillListenGlyph, "listen1", "listen2":
		return ""
	default:
		return ""
	}
}

// RollupSkills 把字下各技能状态收成一个字级状态。
//
// 规则（从严）：
//  1. 空 → not_started
//  2. 任一 shaky → shaky
//  3. 全部为 mastered 或 review_due → 有 review_due 则 review_due，否则 mastered
//  4. 有任何进展但未全过 → learning
//  5. 否则 not_started
func RollupSkills(statuses []Status) Status {
	if len(statuses) == 0 {
		return StatusNotStarted
	}

	anyShaky := false
	anyDue := false
	anyProgress := false
	doneCount := 0

	for _, st := range statuses {
		switch st {
		case StatusShaky:
			anyShaky = true
			anyProgress = true
		case StatusLearning:
			anyProgress = true
		case StatusReviewDue:
			anyDue = true
			anyProgress = true
			doneCount++
		case StatusMastered:
			anyProgress = true
			doneCount++
		}
	}

	if anyShaky {
		return StatusShaky
	}
	if doneCount >= len(statuses) {
		if anyDue {
			return StatusReviewDue
		}
		return StatusMastered
	}
	if anyProgress {
		return StatusLearning
	}
	return StatusNotStarted
}

// IsSkillDone 技能是否算「过关」（掌握或待复习）。
func IsSkillDone(st Status) bool {
	return st == StatusMastered || st == StatusReviewDue
}
