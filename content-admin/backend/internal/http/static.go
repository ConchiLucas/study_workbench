package httpapi

import (
	"embed"
	"io/fs"
	nethttp "net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distFS embed.FS

func mountSPA(r *gin.Engine) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return
	}
	fileServer := nethttp.FileServer(nethttp.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") {
			c.JSON(nethttp.StatusNotFound, gin.H{"error": "接口不存在"})
			return
		}
		if _, err := fs.Stat(sub, strings.TrimPrefix(p, "/")); err != nil {
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
