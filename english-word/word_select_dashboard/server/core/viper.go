package core

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/conchi/go-react-template/server/core/internal"
	"github.com/conchi/go-react-template/server/global"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Viper 配置
func Viper() *viper.Viper {
	config := getConfigPath()

	v := viper.New()
	v.SetConfigFile(config)
	v.SetConfigType("yaml")
	if err := bindRuntimeEnvironment(v); err != nil {
		panic(fmt.Errorf("fatal error bind runtime environment: %w", err))
	}
	err := readConfig(v, config)
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	v.WatchConfig()

	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
		if err = v.Unmarshal(&global.GVA_CONFIG); err != nil {
			fmt.Println(err)
		}
	})
	if err = v.Unmarshal(&global.GVA_CONFIG); err != nil {
		panic(fmt.Errorf("fatal error unmarshal config: %w", err))
	}

	// root 适配性 根据root位置去找到对应迁移位置,保证root路径有效
	return v
}

func bindRuntimeEnvironment(v *viper.Viper) error {
	bindings := map[string]string{
		"pgsql.path":              "SELECT_DB_CONTAINER_HOST",
		"pgsql.port":              "SELECT_DB_PORT",
		"pgsql.db-name":           "SELECT_DB_NAME",
		"pgsql.username":          "SELECT_DB_USER",
		"pgsql.password":          "SELECT_DB_PASSWORD",
		"pgsql.config":            "SELECT_DB_CONFIG",
		"redis.addr":              "REDIS_CONTAINER_ADDR",
		"redis.password":          "REDIS_PASSWORD",
		"minio.endpoint":          "MINIO_CONTAINER_ENDPOINT",
		"minio.access-key-id":     "MINIO_ACCESS_KEY",
		"minio.secret-access-key": "MINIO_SECRET_KEY",
		"minio.bucket-name":       "MINIO_BUCKET",
		"minio.use-ssl":           "MINIO_USE_SSL",
		"word-agent.base-url":     "WORD_AGENT_CONTAINER_URL",
		"system.addr":             "DASHBOARD_SERVER_PORT",
	}
	for key, environment := range bindings {
		if err := v.BindEnv(key, environment); err != nil {
			return fmt.Errorf("bind %s: %w", key, err)
		}
	}
	return nil
}

func readConfig(v *viper.Viper, config string) error {
	if err := v.ReadInConfig(); err != nil {
		return explainConfigReadError(config, err)
	}
	return nil
}

func explainConfigReadError(config string, err error) error {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && os.IsNotExist(pathErr) {
		templatePath := filepath.Join(filepath.Dir(config), "config.template.yaml")
		if _, statErr := os.Stat(templatePath); statErr == nil {
			return fmt.Errorf(
				"open %s: no such file or directory; copy %s to %s and update it before starting",
				config,
				templatePath,
				config,
			)
		}
	}
	return err
}

// getConfigPath 获取配置文件路径, 优先级: 命令行 > 环境变量 > 默认值
func getConfigPath() (config string) {
	// `-c` flag parse
	flag.StringVar(&config, "c", "", "choose config file.")
	flag.Parse()
	if config != "" { // 命令行参数不为空 将值赋值于config
		fmt.Printf("您正在使用命令行的 '-c' 参数传递的值, config 的路径为 %s\n", config)
		return
	}
	if env := os.Getenv(internal.ConfigEnv); env != "" { // 判断环境变量 GVA_CONFIG
		config = env
		fmt.Printf("您正在使用 %s 环境变量, config 的路径为 %s\n", internal.ConfigEnv, config)
		return
	}

	config = internal.ConfigDefaultFile
	fmt.Printf("您正在单系统环境模式运行, config 的默认路径为 %s\n", config)

	return
}
