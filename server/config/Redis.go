package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type RedisConfig struct {
	Enable   bool   `mapstructure:"enable"`
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func LoadRedisConfig() (*RedisConfig, error) {
	v := viper.New()
	v.SetConfigFile("D:\\GoStudy\\ToDoList\\server\\config.yml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}
	var cfg RedisConfig
	if err := v.UnmarshalKey("redis", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal redis failed: %w", err)
	}

	return &cfg, nil
}
