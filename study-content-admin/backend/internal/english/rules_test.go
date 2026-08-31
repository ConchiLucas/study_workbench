package english_test

import (
	"testing"

	"github.com/conchi/study-content-admin/internal/english"
	"github.com/stretchr/testify/assert"
)

func TestNeedsSenseImage(t *testing.T) {
	assert.False(t, english.NeedsSenseImage("hello"))
	assert.False(t, english.NeedsSenseImage("yes"))
	assert.False(t, english.NeedsSenseImage("yesterday"))
	assert.False(t, english.NeedsSenseImage("kind"))
	assert.True(t, english.NeedsSenseImage("cat"))
	assert.True(t, english.NeedsSenseImage("happy"))
	assert.True(t, english.NeedsSenseImage("welcome"))
}
