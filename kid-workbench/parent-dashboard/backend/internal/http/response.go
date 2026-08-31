package http

import "github.com/gin-gonic/gin"

func ok200(c *gin.Context, data any) {
	c.JSON(200, gin.H{"data": data, "error": nil})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"data": nil, "error": msg})
}
