package mastery_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/conchi/study-workbench/internal/mastery"
)

func TestLiteracySkillCodes(t *testing.T) {
	require.Equal(t, []string{
		mastery.SkillGlyphSense,
		mastery.SkillSenseChar,
	}, mastery.LiteracySkills)
}

func TestPinyinSkillCodes(t *testing.T) {
	require.Equal(t, []string{
		mastery.SkillPinyinInWord,
		mastery.SkillPinyinListen,
	}, mastery.PinyinSkills)
	require.Equal(t, mastery.PinyinSkills, mastery.SkillsForSubject("pinyin"))
	require.Equal(t, mastery.LiteracySkills, mastery.SkillsForSubject("literacy"))
	require.Nil(t, mastery.SkillsForSubject("math"))
}

func TestSkillFromQuestionCode(t *testing.T) {
	require.Equal(t, "", mastery.SkillFromQuestionCode("listen_glyph"))
	require.Equal(t, "", mastery.SkillFromQuestionCode("listen1"))
	require.Equal(t, mastery.SkillGlyphSense, mastery.SkillFromQuestionCode("glyph_sense"))
	require.Equal(t, mastery.SkillSenseChar, mastery.SkillFromQuestionCode("sense_char"))
	require.Equal(t, mastery.SkillPinyinInWord, mastery.SkillFromQuestionCode("inword"))
	require.Equal(t, mastery.SkillPinyinListen, mastery.SkillFromQuestionCode("listen"))
	require.Equal(t, "", mastery.SkillFromQuestionCode("calc"))
}

func TestRollupAllMastered(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusMastered,
		mastery.StatusReviewDue,
	})
	require.Equal(t, mastery.StatusReviewDue, got)
}

func TestRollupAllMasteredNoDue(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusMastered,
		mastery.StatusMastered,
	})
	require.Equal(t, mastery.StatusMastered, got)
}

func TestRollupAnyShaky(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusMastered,
		mastery.StatusShaky,
	})
	require.Equal(t, mastery.StatusShaky, got)
}

func TestRollupLearningBeatsNotStarted(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusNotStarted,
		mastery.StatusLearning,
	})
	require.Equal(t, mastery.StatusLearning, got)
}

func TestRollupEmptyIsNotStarted(t *testing.T) {
	require.Equal(t, mastery.StatusNotStarted, mastery.RollupSkills(nil))
}

func TestRollupIncompleteNotFullyMastered(t *testing.T) {
	got := mastery.RollupSkills([]mastery.Status{
		mastery.StatusMastered,
		mastery.StatusNotStarted,
	})
	require.Equal(t, mastery.StatusLearning, got)
}
