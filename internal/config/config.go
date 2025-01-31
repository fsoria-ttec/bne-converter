package config

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus" // logging
	"github.com/spf13/viper"     // config
)

type Config struct {
	Version  string `mapstructure:"version"`
	Database DatabaseConfig
	Crawler  CrawlerConfig
	Monitor  MonitorConfig
	Logging  LoggingConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type CrawlerConfig struct {
	BaseURL                string           `mapstructure:"base_url"`
	CheckInterval          time.Duration    `mapstructure:"check_interval"`
	DownloadPath           string           `mapstructure:"download_path"`
	MaxConcurrentDownloads int              `mapstructure:"max_concurrent_downloads"`
	RetryAttempts          int              `mapstructure:"retry_attempts"`
	RetryDelay             time.Duration    `mapstructure:"retry_delay"`
	Categories             []string         `mapstructure:"categories"`
	ManualMode             ManualModeConfig `mapstructure:"manual_mode"`
}

type ManualModeConfig struct {
	DeleteAfter        bool     `mapstructure:"delete_after"`
	SelectedCategories []string `mapstructure:"selected_categories"`
}

type MonitorConfig struct {
	CheckInterval time.Duration `mapstructure:"check_interval"`
	Timeout       time.Duration `mapstructure:"timeout"`
}

type LoggingConfig struct {
	Level           string
	Format          string
	Output          string
	TimestampFormat string `mapstructure:"timestamp_format"`
}

func (l *LoggingConfig) GetLogLevel() logrus.Level {
	switch l.Level {
	case "panic":
		return logrus.PanicLevel
	case "fatal":
		return logrus.FatalLevel
	case "error":
		return logrus.ErrorLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "info":
		return logrus.InfoLevel
	case "debug":
		return logrus.DebugLevel
	case "trace":
		return logrus.TraceLevel
	default:
		return logrus.InfoLevel
	}
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Error leyendo archivo de configuración: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("Error en el unmarshaling de la configuración")
	}

	return &config, nil
}

func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}
