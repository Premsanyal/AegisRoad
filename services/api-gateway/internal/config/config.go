package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Environment      string
	HTTPPort         int
	GRPCPort         int
	DatabaseURL      string
	RedisURL         string
	KafkaBrokers     string
	RoutingGRPCAddr  string
	JWTSecret        string
	LogLevel         string
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/aegisroad")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("API")

	// Defaults
	viper.SetDefault("environment", "development")
	viper.SetDefault("http_port", 8081)
	viper.SetDefault("grpc_port", 9090)
	viper.SetDefault("database_url", "postgres://aegis:aegisroad_dev@localhost:5432/aegisroad")
	viper.SetDefault("redis_url", "redis://localhost:6379")
	viper.SetDefault("kafka_brokers", "localhost:9092")
	viper.SetDefault("routing_grpc_addr", "localhost:50051")
	viper.SetDefault("jwt_secret", "dev_secret_change_in_production")
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