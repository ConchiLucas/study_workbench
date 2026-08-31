package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	serverConfig "github.com/conchi/go-react-template/server/config"
	"github.com/spf13/viper"
)

func TestReadConfigIncludesTemplateHintWhenDefaultConfigMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	templatePath := filepath.Join(dir, "config.template.yaml")

	if err := os.WriteFile(templatePath, []byte("system:\n  addr: 8008\n"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	err := readConfig(v, configPath)
	if err == nil {
		t.Fatal("expected missing config error")
	}

	if !strings.Contains(err.Error(), "config.template.yaml") {
		t.Fatalf("expected template hint, got %v", err)
	}
}

func TestReadConfigSucceedsWhenConfigExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(configPath, []byte("system:\n  addr: 8008\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	if err := readConfig(v, configPath); err != nil {
		t.Fatalf("expected config read to succeed: %v", err)
	}
}

func TestBindRuntimeEnvironmentOverridesMachineSpecificConfig(t *testing.T) {
	t.Setenv("SELECT_DB_CONTAINER_HOST", "host.docker.internal")
	t.Setenv("SELECT_DB_PORT", "5544")
	t.Setenv("SELECT_DB_NAME", "portable_select")
	t.Setenv("SELECT_DB_USER", "portable_user")
	t.Setenv("SELECT_DB_PASSWORD", "portable_secret")
	t.Setenv("SELECT_DB_CONFIG", "sslmode=require TimeZone=Asia/Shanghai")
	t.Setenv("REDIS_CONTAINER_ADDR", "host.docker.internal:6380")
	t.Setenv("REDIS_PASSWORD", "redis_secret")
	t.Setenv("MINIO_CONTAINER_ENDPOINT", "host.docker.internal:19101")
	t.Setenv("MINIO_ACCESS_KEY", "portable_minio")
	t.Setenv("MINIO_SECRET_KEY", "portable_minio_secret")
	t.Setenv("MINIO_BUCKET", "portable-bucket")
	t.Setenv("MINIO_USE_SSL", "true")
	t.Setenv("WORD_AGENT_CONTAINER_URL", "http://host.docker.internal:6017")
	t.Setenv("DASHBOARD_SERVER_PORT", "6015")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	configText := `
pgsql:
  path: fixed-db
  port: "5432"
  db-name: fixed-name
  username: fixed-user
  password: fixed-password
  config: sslmode=disable
redis:
  addr: fixed-redis:6379
  password: fixed-redis-password
minio:
  endpoint: fixed-minio:9000
  access-key-id: fixed-access
  secret-access-key: fixed-secret
  bucket-name: fixed-bucket
  use-ssl: false
word-agent:
  base-url: http://fixed-agent:8010
system:
  addr: 21417
`
	if err := os.WriteFile(configPath, []byte(configText), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	if err := bindRuntimeEnvironment(v); err != nil {
		t.Fatalf("bind runtime environment: %v", err)
	}
	if err := readConfig(v, configPath); err != nil {
		t.Fatalf("read config: %v", err)
	}
	var resolved serverConfig.Server
	if err := v.Unmarshal(&resolved); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	if resolved.Pgsql.Path != "host.docker.internal" || resolved.Pgsql.Port != "5544" {
		t.Fatalf("unexpected pgsql endpoint: %#v", resolved.Pgsql)
	}
	if resolved.Pgsql.Dbname != "portable_select" || resolved.Pgsql.Username != "portable_user" {
		t.Fatalf("unexpected pgsql identity: %#v", resolved.Pgsql)
	}
	if resolved.Pgsql.Password != "portable_secret" || resolved.Pgsql.Config != "sslmode=require TimeZone=Asia/Shanghai" {
		t.Fatalf("unexpected pgsql credentials/options")
	}
	if resolved.Redis.Addr != "host.docker.internal:6380" || resolved.Redis.Password != "redis_secret" {
		t.Fatalf("unexpected redis config: %#v", resolved.Redis)
	}
	if resolved.Minio.Endpoint != "host.docker.internal:19101" || resolved.Minio.BucketName != "portable-bucket" || !resolved.Minio.UseSSL {
		t.Fatalf("unexpected minio config: %#v", resolved.Minio)
	}
	if resolved.Minio.AccessKeyID != "portable_minio" || resolved.Minio.SecretAccessKey != "portable_minio_secret" {
		t.Fatalf("unexpected minio credentials")
	}
	if resolved.WordAgent.BaseURL != "http://host.docker.internal:6017" {
		t.Fatalf("unexpected word agent URL: %s", resolved.WordAgent.BaseURL)
	}
	if resolved.System.Addr != 6015 {
		t.Fatalf("unexpected server port: %d", resolved.System.Addr)
	}
}
