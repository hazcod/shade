package config

import (
	"fmt"
	"github.com/asaskevich/govalidator"
	"gopkg.in/yaml.v3"
	"os"
)

const (
	defaultPort     = 8080
	defaultLogLevel = "info"
)

type Config struct {
	Log struct {
		Level string `yaml:"level" env:"LOG_LEVEL"`
	} `yaml:"log"`

	HTTP struct {
		Port      uint16 `yaml:"port" env:"HTTP_PORT"`
		Interface string `yaml:"interface" env:"HTTP_INTERFACE"`
		Origin    string `yaml:"origin" env:"HTTP_ORIGIN"`
		TLS       struct {
			Certificate string `yaml:"certificate" env:"HTTP_TLS_CERTIFICATE"`
			Key         string `yaml:"key" env:"HTTP_TLS_KEY"`
		} `yaml:"tls"`
	} `yaml:"http"`

	Storage struct {
		Type       string            `yaml:"type" env:"STORAGE_TYPE"`
		Properties map[string]string `yaml:"properties" env:"STORAGE_PROPERTIES"`
	} `yaml:"storage"`

	Auth struct {
		Type       string                 `yaml:"type" env:"AUTH_TYPE"`
		Secret     string                 `yaml:"secret" env:"AUTH_SECRET"`
		Properties map[string]interface{} `yaml:"properties" env:"AUTH_PROPERTIES"`
	} `yaml:"auth"`
}

func LoadConfig(cfgPath string) (*Config, error) {
	cfg := Config{}

	if cfgPath != "" {
		yamlBytes, err := os.ReadFile(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}

		if err := yaml.Unmarshal(yamlBytes, &cfg); err != nil {
			return nil, fmt.Errorf("error parsing config file: %w", err)
		}
	}

	if cfg.HTTP.Port == 0 {
		cfg.HTTP.Port = defaultPort
	}

	if cfg.Log.Level == "" {
		cfg.Log.Level = defaultLogLevel
	}

	if cfg.Auth.Secret == "" {
		return nil, fmt.Errorf("auth secret is required")
	}

	if cfg.HTTP.Origin == "" {
		httpPrefix := "http"
		if cfg.HTTP.TLS.Key != "" {
			httpPrefix += "s"
		}
		cfg.HTTP.Origin = fmt.Sprintf("%s://%s:%d", httpPrefix, cfg.HTTP.Interface, cfg.HTTP.Port)
	}

	valid, err := govalidator.ValidateStruct(&cfg)
	if !valid || err != nil {
		return nil, fmt.Errorf("error validating config: %w", err)
	}

	return &cfg, nil
}
