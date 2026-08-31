package config

type Server struct {
	JWT       JWT       `mapstructure:"jwt" json:"jwt" yaml:"jwt"`
	Zap       Zap       `mapstructure:"zap" json:"zap" yaml:"zap"`
	Redis     Redis     `mapstructure:"redis" json:"redis" yaml:"redis"`
	RedisList []Redis   `mapstructure:"redis-list" json:"redis-list" yaml:"redis-list"`
	Mongo     Mongo     `mapstructure:"mongo" json:"mongo" yaml:"mongo"`
	System    System    `mapstructure:"system" json:"system" yaml:"system"`
	AI        AI        `mapstructure:"ai" json:"ai" yaml:"ai"`
	WordAgent WordAgent `mapstructure:"word-agent" json:"word-agent" yaml:"word-agent"`
	// gorm
	Mysql  Mysql           `mapstructure:"mysql" json:"mysql" yaml:"mysql"`
	Mssql  Mssql           `mapstructure:"mssql" json:"mssql" yaml:"mssql"`
	Pgsql  Pgsql           `mapstructure:"pgsql" json:"pgsql" yaml:"pgsql"`
	Oracle Oracle          `mapstructure:"oracle" json:"oracle" yaml:"oracle"`
	Sqlite Sqlite          `mapstructure:"sqlite" json:"sqlite" yaml:"sqlite"`
	DBList []SpecializedDB `mapstructure:"db-list" json:"db-list" yaml:"db-list"`

	// 跨域配置
	Cors CORS `mapstructure:"cors" json:"cors" yaml:"cors"`

	// MCP配置
	MCP MCP `mapstructure:"mcp" json:"mcp" yaml:"mcp"`

	// MinIO配置
	Minio Minio `mapstructure:"minio" json:"minio" yaml:"minio"`
}
