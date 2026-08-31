package literacy_test

import (
	"testing"

	"github.com/conchi/study-content-admin/internal/literacy"
	"github.com/stretchr/testify/assert"
)

func TestNeedsSenseImage(t *testing.T) {
	assert.False(t, literacy.NeedsSenseImage("的"))
	assert.False(t, literacy.NeedsSenseImage("这"))
	assert.True(t, literacy.NeedsSenseImage("我"))
	assert.False(t, literacy.NeedsSenseImage("是"))
	assert.True(t, literacy.NeedsSenseImage("人"))
	assert.True(t, literacy.NeedsSenseImage("口"))
	assert.True(t, literacy.NeedsSenseImage("火"))
	assert.True(t, literacy.NeedsSenseImage("一"))
	assert.True(t, literacy.NeedsSenseImage("对"))
	assert.True(t, literacy.NeedsSenseImage("给"))
	assert.True(t, literacy.NeedsSenseImage("被"))
}
