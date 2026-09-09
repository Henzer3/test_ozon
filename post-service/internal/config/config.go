package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel  string `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	DBAddress string `yaml:"db_address" env:"DB_ADDRESS" env-required:"true"`
	AppID     int    `yaml:"app_id" env:"APP_ID" env-default:"1"`

	SSO        SSOConfig  `yaml:"sso"`
	HTTPConfig HTTPConfig `yaml:"http"`
}

type SSOConfig struct {
	SSOAddress string `yaml:"sso_address" env:"SSO_ADDRESS" env-required:"true"`
}

type HTTPConfig struct {
	Address              string        `yaml:"address" env:"API_ADDRESS" env-required:"true"`
	Timeout              time.Duration `yaml:"timeout" env:"API_TIMEOUT" env-default:"5s"`
	GracefulShutdownTime time.Duration `yaml:"graceful_shutdown_time" env:"GRACEFUL_SHUTDOWN_TIME" env-default:"3s"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
