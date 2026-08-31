package utils

import (
	"context"
	"io"
	"mime/multipart"
	"net/url"
	"path"
	"strings"

	"github.com/conchi/go-react-template/server/global"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

var MinioClient *minio.Client

func InitMinio() {
	cfg := global.GVA_CONFIG.Minio
	if cfg.Endpoint == "" || cfg.BucketName == "" {
		global.GVA_LOG.Info("minio config empty, skipping init")
		return
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		global.GVA_LOG.Error("minio init failed", zap.Error(err))
		return
	}
	MinioClient = client

	// 检查 Bucket 是否存在，不存在则自动创建
	exists, errBucketExists := MinioClient.BucketExists(context.Background(), cfg.BucketName)
	if errBucketExists == nil && !exists {
		err = MinioClient.MakeBucket(context.Background(), cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			global.GVA_LOG.Error("minio create bucket failed", zap.Error(err))
		}
		// 默认公开访问权限策略
		policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::` + cfg.BucketName + `/*"]}]}`
		MinioClient.SetBucketPolicy(context.Background(), cfg.BucketName, policy)
	}

	global.GVA_LOG.Info("minio init success")
}

// UploadToMinio uploads a generic file stream and returns the url
func UploadToMinio(file *multipart.FileHeader, objectPrefix string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	return UploadReaderToMinio(src, file.Size, file.Filename, file.Header.Get("Content-Type"), objectPrefix)
}

func UploadReaderToMinio(reader io.Reader, size int64, originalName, contentType, objectPrefix string) (string, error) {
	cfg := global.GVA_CONFIG.Minio
	objectName := path.Join(cfg.BasePath, objectPrefix, originalName)

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := MinioClient.PutObject(context.Background(), cfg.BucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	// 返回相对可用链接
	u := url.URL{Path: objectName}
	return "/" + cfg.BucketName + "/" + u.EscapedPath(), nil
}

// DeleteFromMinio deletes an object based on the relative object name
func DeleteFromMinio(objectName string) error {
	cfg := global.GVA_CONFIG.Minio

	// 兼容 HTTP 完整前缀
	if strings.HasPrefix(objectName, "http://") || strings.HasPrefix(objectName, "https://") {
		prefix := cfg.Endpoint + "/" + cfg.BucketName + "/"
		if idx := strings.Index(objectName, prefix); idx != -1 {
			objectName = objectName[idx+len(prefix):]
		}
	}

	// 处理相对前缀
	relativePrefix := "/" + cfg.BucketName + "/"
	if strings.HasPrefix(objectName, relativePrefix) {
		objectName = objectName[len(relativePrefix):]
	}

	if unescaped, err := url.PathUnescape(objectName); err == nil {
		objectName = unescaped
	}

	return MinioClient.RemoveObject(context.Background(), cfg.BucketName, objectName, minio.RemoveObjectOptions{})
}
