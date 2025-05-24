package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Webhook      WebhookConfig      `yaml:"webhook"`
	Database     DatabaseConfig     `yaml:"database"`
	Logging      LoggingConfig      `yaml:"logging"`
	Deployments  []DeploymentConfig `yaml:"deployments"`
	Notification NotificationConfig `yaml:"notifications"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

type WebhookConfig struct {
	Secret string `yaml:"secret"`
	Path   string `yaml:"path"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

type DeploymentConfig struct {
	Name       string   `yaml:"name"`
	Repository string   `yaml:"repository"`
	Branch     string   `yaml:"branch"`
	WorkDir    string   `yaml:"work_dir"`
	Commands   []string `yaml:"commands"`
}

type NotificationConfig struct {
	Webhook WebhookNotificationConfig `yaml:"webhook"`
	Email   EmailNotificationConfig   `yaml:"email"`
}

type WebhookNotificationConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
}

type EmailNotificationConfig struct {
	Enabled  bool     `yaml:"enabled"`
	SMTPHost string   `yaml:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	To       []string `yaml:"to"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) GetDeploymentConfig(repository string) *DeploymentConfig {
	for _, deploy := range c.Deployments {
		if deploy.Repository == repository {
			return &deploy
		}
	}
	return nil
}
