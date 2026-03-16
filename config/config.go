package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	URL   string `mapstructure:"url"`
	Token string `mapstructure:"token"`
}

func Load() (*Config, error) {
	viper.SetEnvPrefix("VIKUNJA")
	viper.AutomaticEnv()

	// Config file: ~/.config/vikunja-cli/config.yaml
	configDir, err := os.UserConfigDir()
	if err == nil {
		viper.AddConfigPath(filepath.Join(configDir, "vikunja-cli"))
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	_ = viper.ReadInConfig() // ignore if not found

	cfg := &Config{
		URL:   viper.GetString("url"),
		Token: viper.GetString("token"),
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("vikunja URL not set. Use VIKUNJA_URL env var or set 'url' in ~/.config/vikunja-cli/config.yaml")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("vikunja token not set. Use VIKUNJA_TOKEN env var or set 'token' in ~/.config/vikunja-cli/config.yaml")
	}

	return cfg, nil
}
