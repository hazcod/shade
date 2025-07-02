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
	HTTP struct {
		Port      uint16 `yaml:"port" env:"HTTP_PORT"`
		Interface string `yaml:"interface" env:"HTTP_INTERFACE"`
	} `yaml:"http"`

	Storage struct {
		Type       string            `yaml:"type" env:"STORAGE_TYPE"`
		Properties map[string]string `yaml:"properties" env:"STORAGE_PROPERTIES"`
	} `yaml:"storage"`

	Log struct {
		Level string `yaml:"level" env:"LOG_LEVEL"`
	} `yaml:"log"`
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

	valid, err := govalidator.ValidateStruct(&cfg)
	if !valid || err != nil {
		return nil, fmt.Errorf("error validating config: %w", err)
	}

	return &cfg, nil
}
