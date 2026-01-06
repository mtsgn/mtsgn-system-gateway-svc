package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func LoadConfig(path string) (*Config, error) {
	conf := Config{}
	viper.SetConfigFile(path)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	err = viper.Unmarshal(&conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &conf, nil
}
