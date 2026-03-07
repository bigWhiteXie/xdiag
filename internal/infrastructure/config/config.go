package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	ConfigDirName     = ".xdiag"
	ConfigFileName    = "config"
	ConfigFileType    = "yaml"
	
	// LLM Config Keys
	LLMAPIKey    = "llm.api_key"
	LLMBaseURL   = "llm.base_url"
	LLMProtocol  = "llm.protocol"
	LLMModelName = "llm.model_name"
	
	// Error messages
	ErrorLoadingConfig = "error loading config file: %w"
	ErrorSavingConfig  = "error saving config: %w"
	
	// Success messages
	SuccessConfigSaved = "config saved to %s"
)

// Config 应用程序配置
type Config struct {
	LLM LLMConfig `mapstructure:"llm" json:"llm"`
}

// LLMConfig LLM相关配置
type LLMConfig struct {
	APIKey    string `mapstructure:"api_key" json:"api_key"`
	BaseURL   string `mapstructure:"base_url" json:"base_url"`
	Protocol  string `mapstructure:"protocol" json:"protocol"`
	ModelName string `mapstructure:"model_name" json:"model_name"`
}

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ConfigDirName)
	viper.AddConfigPath(configPath)
	viper.SetConfigType(ConfigFileType)
	viper.SetConfigName(ConfigFileName)

	// 设置默认值
	viper.SetDefault(LLMProtocol, "openai")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf(ErrorLoadingConfig, err)
		}
		// 如果配置文件不存在，继续使用默认值
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

// SaveConfig 保存配置
func SaveConfig(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ConfigDirName)
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(configPath, ConfigFileName+"."+ConfigFileType)

	// 创建临时 viper 实例来保存配置
	v := viper.New()
	v.Set(LLMAPIKey, cfg.LLM.APIKey)
	v.Set(LLMBaseURL, cfg.LLM.BaseURL)
	v.Set(LLMProtocol, cfg.LLM.Protocol)
	v.Set(LLMModelName, cfg.LLM.ModelName)

	if err := v.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf(ErrorSavingConfig, err)
	}

	fmt.Printf(SuccessConfigSaved+"\n", configFile)
	return nil
}