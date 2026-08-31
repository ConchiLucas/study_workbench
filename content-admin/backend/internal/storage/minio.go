package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/conchi/study-content-admin/internal/configclient"
)

type ObjectStore struct {
	client   *minio.Client
	bucket   string
	basePath string
}

func NewFromConfig(cfg configclient.ObjectStorageConfiguration) (*ObjectStore, error) {
	if !cfg.Configured || !cfg.Enabled {
		return nil, fmt.Errorf("MinIO 未配置或未启用，请在配置中心设置")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if v := os.Getenv("APP_MINIO_ENDPOINT"); v != "" {
		endpoint = v
	}
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &ObjectStore{
		client:   client,
		bucket:   cfg.BucketName,
		basePath: strings.Trim(cfg.BasePath, "/"),
	}, nil
}

func (s *ObjectStore) fullKey(relative string) string {
	if s.basePath == "" {
		return relative
	}
	return path.Join(s.basePath, relative)
}

func (s *ObjectStore) resolveKey(relativeOrFullKey string) string {
	key := relativeOrFullKey
	if s.basePath != "" && !strings.HasPrefix(relativeOrFullKey, s.basePath+"/") {
		key = s.fullKey(relativeOrFullKey)
	}
	return key
}

// PutBytes uploads bytes and returns the full object key (not a public URL).
func (s *ObjectStore) PutBytes(ctx context.Context, relativeKey string, data []byte, contentType string) (string, error) {
	key := s.fullKey(relativeKey)
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *ObjectStore) GetBytes(ctx context.Context, relativeOrFullKey string) ([]byte, error) {
	key := s.resolveKey(relativeOrFullKey)
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

// PutPNG uploads bytes and returns the full object key (not a public URL).
func (s *ObjectStore) PutPNG(ctx context.Context, relativeKey string, png []byte) (string, error) {
	return s.PutBytes(ctx, relativeKey, png, "image/png")
}

func (s *ObjectStore) GetPNG(ctx context.Context, relativeOrFullKey string) ([]byte, error) {
	return s.GetBytes(ctx, relativeOrFullKey)
}

func GlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("literacy/glyphs/%d.png", kpID)
}

func SenseObjectKey(kpID int64) string {
	return fmt.Sprintf("literacy/senses/%d.png", kpID)
}

func SpeechObjectKey(kpID int64) string {
	return fmt.Sprintf("literacy/speech/%d.mp3", kpID)
}

func (s *ObjectStore) GlyphKey(kpID int64) string {
	return s.fullKey(GlyphObjectKey(kpID))
}

func (s *ObjectStore) SenseKey(kpID int64) string {
	return s.fullKey(SenseObjectKey(kpID))
}

func (s *ObjectStore) SpeechKey(kpID int64) string {
	return s.fullKey(SpeechObjectKey(kpID))
}

func PinyinSoloObjectKey(kpID int64) string {
	return fmt.Sprintf("pinyin/speech/%d/solo.mp3", kpID)
}

func PinyinWordObjectKey(kpID int64) string {
	return fmt.Sprintf("pinyin/speech/%d/word.mp3", kpID)
}

func (s *ObjectStore) PinyinSoloKey(kpID int64) string {
	return s.fullKey(PinyinSoloObjectKey(kpID))
}

func (s *ObjectStore) PinyinWordKey(kpID int64) string {
	return s.fullKey(PinyinWordObjectKey(kpID))
}

func PinyinGlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("pinyin/glyphs/%d.png", kpID)
}

func (s *ObjectStore) PinyinGlyphKey(kpID int64) string {
	return s.fullKey(PinyinGlyphObjectKey(kpID))
}

func MathGlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("math/glyphs/%d.png", kpID)
}

func (s *ObjectStore) MathGlyphKey(kpID int64) string {
	return s.fullKey(MathGlyphObjectKey(kpID))
}

func MathSpeechObjectKey(kpID int64) string {
	return fmt.Sprintf("math/speech/%d.mp3", kpID)
}

func (s *ObjectStore) MathSpeechKey(kpID int64) string {
	return s.fullKey(MathSpeechObjectKey(kpID))
}

func EnglishGlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("english/glyphs/%d.png", kpID)
}

func EnglishSenseObjectKey(kpID int64) string {
	return fmt.Sprintf("english/senses/%d.png", kpID)
}

func EnglishSpeechObjectKey(kpID int64) string {
	return fmt.Sprintf("english/speech/%d.mp3", kpID)
}

func (s *ObjectStore) EnglishGlyphKey(kpID int64) string {
	return s.fullKey(EnglishGlyphObjectKey(kpID))
}

func (s *ObjectStore) EnglishSenseKey(kpID int64) string {
	return s.fullKey(EnglishSenseObjectKey(kpID))
}

func (s *ObjectStore) EnglishSpeechKey(kpID int64) string {
	return s.fullKey(EnglishSpeechObjectKey(kpID))
}

func ScienceGlyphObjectKey(kpID int64) string {
	return fmt.Sprintf("science/glyphs/%d.png", kpID)
}

func ScienceSenseObjectKey(kpID int64) string {
	return fmt.Sprintf("science/senses/%d.png", kpID)
}

func ScienceSpeechObjectKey(kpID int64) string {
	return fmt.Sprintf("science/speech/%d.mp3", kpID)
}

func (s *ObjectStore) ScienceGlyphKey(kpID int64) string {
	return s.fullKey(ScienceGlyphObjectKey(kpID))
}

func (s *ObjectStore) ScienceSenseKey(kpID int64) string {
	return s.fullKey(ScienceSenseObjectKey(kpID))
}

func (s *ObjectStore) ScienceSpeechKey(kpID int64) string {
	return s.fullKey(ScienceSpeechObjectKey(kpID))
}
