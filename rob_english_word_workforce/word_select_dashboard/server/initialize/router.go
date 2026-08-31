package initialize

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/conchi/go-react-template/server/global"
	"github.com/conchi/go-react-template/server/middleware"
	"github.com/conchi/go-react-template/server/router"
	"github.com/gin-gonic/gin"
)

type justFilesFilesystem struct {
	fs http.FileSystem
}

func (fs justFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, os.ErrPermission
	}

	return f, nil
}

func Routers() *gin.Engine {
	Router := gin.New()
	Router.MaxMultipartMemory = 50 << 20 // 50 MB
	Router.Use(middleware.GinRecovery(true))
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}

	systemRouter := router.RouterGroupApp.System

	Router.Use(middleware.CorsByRules())

	// MinIO 反向代理：将 /<bucket-name>/* 请求代理到 MinIO 服务
	// 解决通过 frp 外网访问时图片无法显示的问题
	minioCfg := global.GVA_CONFIG.Minio
	if minioCfg.Endpoint != "" && minioCfg.BucketName != "" {
		scheme := "http"
		if minioCfg.UseSSL {
			scheme = "https"
		}
		minioTarget, err := url.Parse(fmt.Sprintf("%s://%s", scheme, minioCfg.Endpoint))
		if err == nil {
			minioProxy := httputil.NewSingleHostReverseProxy(minioTarget)
			minioHandler := func(c *gin.Context) {
				minioProxy.ServeHTTP(c.Writer, c.Request)
			}
			Router.GET("/"+minioCfg.BucketName+"/*filepath", minioHandler)
			Router.HEAD("/"+minioCfg.BucketName+"/*filepath", minioHandler)
			global.GVA_LOG.Info("MinIO reverse proxy registered: /" + minioCfg.BucketName + "/*")
		}
	}

	PublicGroup := Router.Group(global.GVA_CONFIG.System.RouterPrefix)
	PrivateGroup := Router.Group(global.GVA_CONFIG.System.RouterPrefix)

	PrivateGroup.Use(middleware.JWTAuth())

	{
		PublicGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, "ok")
		})
	}
	{
		systemRouter.InitBaseRouter(PublicGroup)
		systemRouter.InitInitRouter(PublicGroup)
		systemRouter.InitAIConfigRouter(PublicGroup, PrivateGroup)
		systemRouter.InitExecutionConfigRouter(PublicGroup, PrivateGroup)
		systemRouter.InitTTSConfigRouter(PublicGroup, PrivateGroup)
		systemRouter.InitSentenceRouter(PublicGroup)
		systemRouter.InitExecutionRouter(PublicGroup)
		systemRouter.InitClozeResultRouter(PublicGroup)
		systemRouter.InitWordLibraryRouter(PublicGroup)
		systemRouter.InitAppUserRouter(PublicGroup)
	}

	{
		systemRouter.InitUserRouter(PrivateGroup)
	}

	initExtraRouter(PrivateGroup, PublicGroup)

	global.GVA_ROUTERS = Router.Routes()

	RegisterFrontendRoutes(Router)

	global.GVA_LOG.Info("router register success")
	return Router
}
