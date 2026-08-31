package http

import (
	"github.com/gin-gonic/gin"
)

func (h *handlers) listRewards(c *gin.Context) {
	childID, okp := pathInt64(c, "cid")
	if !okp {
		return
	}
	data, err := h.deps.Reward.List(childID)
	if err != nil {
		fail(c, 500, err.Error())
		return
	}
	ok200(c, data)
}

func (h *handlers) redeemReward(c *gin.Context) {
	childID, ok1 := pathInt64(c, "cid")
	rewardID, ok2 := pathInt64(c, "rewardId")
	if !ok1 || !ok2 {
		return
	}
	if err := h.deps.Reward.Redeem(childID, rewardID); err != nil {
		fail(c, 400, err.Error())
		return
	}
	ok200(c, gin.H{"ok": true})
}
