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

// BindEnvs binds every overridable config key to an environment variable so the
// gateway can be configured entirely from the environment (.env) — useful for
// deploy targets that only accept env vars (e.g. Timeweb App Platform). Explicit
// BindEnv is required because viper's AutomaticEnv does not populate nested keys
// during Unmarshal when the key is absent from the config file.
func BindEnvs() error {
	binds := map[string]string{
		"http.host":         "HTTP_HOST",
		"http.port":         "HTTP_PORT",
		"analyzer.address":  "ANALYZER_ADDRESS",
		"connector.address": "CONNECTOR_ADDRESS",
		"app.log_level":     "LOG_LEVEL",
	}

	for key, env := range binds {
		err := viper.BindEnv(key, env)
		if err != nil {
			return fmt.Errorf("bind env %s: %w", env, err)
		}
	}

	return nil
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
