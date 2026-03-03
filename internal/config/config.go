package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the main configuration structure
type Config struct {
	LLM LLMConfig `mapstructure:"llm"`
}

// LLMConfig represents the LLM configuration
type LLMConfig struct {
	APIKey    string `mapstructure:"api_key"`
	BaseURL   string `mapstructure:"base_url"`
	Protocol  string `mapstructure:"protocol"`
	ModelName string `mapstructure:"model_name"`
}

// LoadConfig loads the configuration from the config file
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Join("$HOME", ".xdiag"))

	// 绑定环境变量
	viper.BindEnv("llm.api_key", "XDIAG_API_KEY")
	viper.BindEnv("llm.base_url", "XDIAG_BASE_URL")
	viper.BindEnv("llm.model_name", "XDIAG_MODEL_NAME")

	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，返回默认配置
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return &Config{
				LLM: LLMConfig{
					APIKey:    "",
					BaseURL:   "https://api.openai.com/v1",
					Protocol:  "openai",
					ModelName: "",
				},
			}, nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &cfg, nil
}