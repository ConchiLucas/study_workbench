package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const ctxUserID = "user_id"

// currentUser：v1 免登录，固定注入 userID=1。M7 换成解析 JWT 即可，下游不用改。
func currentUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ctxUserID, int64(1))
		c.Next()
	}
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func pathInt64(c *gin.Context, key string) (int64, bool) {
	v, err := strconv.ParseInt(c.Param(key), 10, 64)
	if err != nil {
		fail(c, 400, "参数 "+key+" 不合法")
		return 0, false
	}
	return v, true
}
