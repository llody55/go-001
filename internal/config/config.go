package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 全局配置。文件内只允许出现占位符，真实口令一律走环境变量注入。
type Config struct {
	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		DSN           string `yaml:"dsn"`
		EnableReplica bool   `yaml:"enable_replica"`
	} `yaml:"database"`
	JWT struct {
		Secret      string `yaml:"secret"`
		ExpireHours int    `yaml:"expire_hours"`
	} `yaml:"jwt"`
}

// Load 读取 configPath（默认 configs/config.yaml），环境变量覆盖敏感项：
// APP_SERVER_PORT / APP_DB_DSN / APP_JWT_SECRET。
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg := &Config{}
	if raw, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(raw, cfg); err != nil {
			return nil, err
		}
	}
	if v := os.Getenv("APP_SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("APP_DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("APP_JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 18080
	}
	if cfg.JWT.ExpireHours == 0 {
		cfg.JWT.ExpireHours = 12
	}
	return cfg, nil
}
