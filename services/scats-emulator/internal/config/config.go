package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Environment       string
	HTTPPort          int
	RedisAddr         string
	KafkaBrokers      string
	LogLevel          string
	IntersectionConfigs []IntersectionConfig
}

type IntersectionConfig struct {
	ID          string
	Name        string
	Latitude    float64
	Longitude   float64
	PhasePlan   models.PhasePlan
	ControllerType string
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/aegisroad")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("SCATS")

	viper.SetDefault("environment", "development")
	viper.SetDefault("http_port", 8082)
	viper.SetDefault("redis_addr", "localhost:6379")
	viper.SetDefault("kafka_brokers", "localhost:9092")
	viper.SetDefault("log_level", "info")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}