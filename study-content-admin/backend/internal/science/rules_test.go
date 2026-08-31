package science_test

import (
	"testing"

	"github.com/conchi/study-content-admin/internal/science"
	"github.com/stretchr/testify/assert"
)

func TestNeedsSenseImageAlwaysTrue(t *testing.T) {
	assert.True(t, science.NeedsSenseImage("冬眠"))
	assert.True(t, science.NeedsSenseImage("光合作用"))
	assert.True(t, science.NeedsSenseImage(""))
}
