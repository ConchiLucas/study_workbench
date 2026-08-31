package config

type Minio struct {
	Endpoint        string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `mapstructure:"access-key-id" json:"access-key-id" yaml:"access-key-id"`
	SecretAccessKey string `mapstructure:"secret-access-key" json:"secret-access-key" yaml:"secret-access-key"`
	UseSSL          bool   `mapstructure:"use-ssl" json:"use-ssl" yaml:"use-ssl"`
	BucketName      string `mapstructure:"bucket-name" json:"bucket-name" yaml:"bucket-name"`
	BasePath        string `mapstructure:"base-path" json:"base-path" yaml:"base-path"`
}
