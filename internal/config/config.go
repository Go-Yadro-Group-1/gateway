package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	DB  DBConfig  `mapstructure:"db"  validate:"required"`
	App AppConfig `mapstructure:"app" validate:"required"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"     validate:"required,hostname|ip"`
	Port     int    `mapstructure:"port"     validate:"required,min=1,max=65535"`
	User     string `mapstructure:"user"     validate:"required,min=1"`
	Password string `mapstructure:"password" validate:"required,min=1"`
	DBName   string `mapstructure:"dbname"   validate:"required,min=1"`
	SSLMode  string `mapstructure:"sslmode"  validate:"required,oneof=disable require verify-ca verify-full"`
}

type AppConfig struct {
	LogLevel string `mapstructure:"log_level" validate:"required,oneof=debug info warn error"`
}

// nolint: gochecknoglobals
var (
	appConfig *Config
	validate  *validator.Validate
)

// nolint: gochecknoinits
func init() {
	validate = validator.New()
}

func LoadConfig() (*Config, error) {
	var cfg Config

	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	err = validate.Struct(&cfg)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	err = cfg.customValidate()
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	appConfig = &cfg

	return &cfg, nil
}

func GetConfig() *Config {
	if appConfig == nil {
		panic("config not loaded")
	}

	return appConfig
}

func (c *Config) customValidate() error {
	return nil
}

func (d *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}
