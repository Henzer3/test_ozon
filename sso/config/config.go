package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel  string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	DBAddress string        `yaml:"db_address" env:"DB_ADDRESS" env-required:"true"`
	TokenTTL  time.Duration `yaml:"token_ttl" env:"TOKEN_TTL" env-default:"1h"`

	GRPC  GRPCConfig  `yaml:"grpc"`
	App   AppConfig   `yaml:"app"`
	Admin AdminConfig `yaml:"admin"`
}

type GRPCConfig struct {
	Address string        `yaml:"address" env:"SSO_ADDRESS" env-default:"localhost:50051"`
	Timeout time.Duration `yaml:"timeout" env:"SSO_TIMEOUT" env-default:"10s"`
}

type AppConfig struct {
	ID     int    `yaml:"id" env:"SSO_APP_ID" env-required:"true"`
	Name   string `yaml:"name" env:"SSO_APP_NAME" env-required:"true"`
	Secret string `yaml:"secret" env:"SSO_APP_SECRET" env-required:"true"`
}

type AdminConfig struct {
	Email    string `yaml:"email" env:"SSO_ADMIN_EMAIL" env-required:"true"`
	Password string `yaml:"password" env:"SSO_ADMIN_PASSWORD" env-required:"true"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
