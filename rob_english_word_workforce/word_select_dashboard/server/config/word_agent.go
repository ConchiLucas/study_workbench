package config

import (
	"strings"
	"time"
)

type WordAgent struct {
	BaseURL        string `mapstructure:"base-url" json:"base-url" yaml:"base-url"`
	TimeoutSeconds int    `mapstructure:"timeout-seconds" json:"timeout-seconds" yaml:"timeout-seconds"`
}

func (w WordAgent) ResolveBaseURL() string {
	baseURL := strings.TrimRight(strings.TrimSpace(w.BaseURL), "/")
	if baseURL == "" {
		return "http://127.0.0.1:8010"
	}
	return baseURL
}

func (w WordAgent) Timeout() time.Duration {
	if w.TimeoutSeconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(w.TimeoutSeconds) * time.Second
}
