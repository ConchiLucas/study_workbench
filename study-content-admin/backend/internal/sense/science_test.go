package sense_test

import (
	"testing"

	"github.com/conchi/study-content-admin/internal/sense"
	"github.com/stretchr/testify/require"
)

func TestPromptScienceNoHanInOutputWhenMapped(t *testing.T) {
	p := sense.PromptScience("冬眠")
	require.NotEmpty(t, p)
	require.Contains(t, p, "child-friendly")
	require.Contains(t, p, "sticker")
	require.False(t, sense.ContainsHan(p), "mapped title 冬眠 leaked Han: %s", p)

	mapped := []string{"冬眠", "光合作用", "种子", "彩虹", "心脏", "垃圾分类", "磁铁"}
	for _, title := range mapped {
		p := sense.PromptScience(title)
		require.NotEmpty(t, p)
		require.Contains(t, p, "child-friendly")
		require.Contains(t, p, "sticker")
		require.False(t, sense.ContainsHan(p), "mapped title %s leaked Han: %s", title, p)
	}

	unmapped := sense.PromptScience("量子纠缠")
	require.NotEmpty(t, unmapped)
	require.Contains(t, unmapped, "child-friendly")
	require.Contains(t, unmapped, "sticker")
}
