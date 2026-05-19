package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	HTTP      HTTPConfig     `mapstructure:"http"      validate:"required"`
	Analyzer  UpstreamConfig `mapstructure:"analyzer"  validate:"required"`
	Connector UpstreamConfig `mapstructure:"connector" validate:"required"`
	App       AppConfig      `mapstructure:"app"       validate:"required"`
}

type HTTPConfig struct {
	Host string `mapstructure:"host" validate:"required,hostname|ip"`
	Port int    `mapstructure:"port" validate:"required,min=1,max=65535"`
}

type UpstreamConfig struct {
	Address string `mapstructure:"address" validate:"required,hostname_port"`
}

type AppConfig struct {
	LogLevel string `mapstructure:"log_level" validate:"required,oneof=debug info warn error"`
}

func LoadConfig() (*Config, error) {
	var cfg Config

	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	err = validator.New().Struct(&cfg)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &cfg, nil
}
