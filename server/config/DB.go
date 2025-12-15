package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type MySQLConfig struct {
	Path            string `mapstructure:"path"`
	Port            int    `mapstructure:"port"`
	Config          string `mapstructure:"config"`
	DBName          string `mapstructure:"db-name"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	MaxIdleConns    int    `mapstructure:"max-idle-conns"`
	MaxOpenConns    int    `mapstructure:"max-open-conns"`
	ConnMaxLifetime string `mapstructure:"conn-max-lifetime"`
	ConnMaxIdleTime string `mapstructure:"conn-max-idle-time"`
}

func LoadMysqlConfig() (string, error) {
	v := viper.New()
	v.SetConfigFile("D:\\GoStudy\\ToDoList\\server\\config.yml")
	if err := v.ReadInConfig(); err != nil {
		return "", fmt.Errorf("read config failed: %w", err)
	}
	var cfg MySQLConfig
	if err := v.UnmarshalKey("mysql", &cfg); err != nil {
		return "", fmt.Errorf("unmarshal mysql failed: %w", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		cfg.Username, cfg.Password, cfg.Path, cfg.Port, cfg.DBName, cfg.Config,
	)

	return dsn, nil
}
