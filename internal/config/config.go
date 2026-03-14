package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bigWhiteXie/xdiag/internal/llm"
	"github.com/bigWhiteXie/xdiag/pkg/logger"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	xdiagDirEnv = "XDIAG_DIR"
)

var (
	keyMap = map[string]string{
		"api_key":    "llm.api_key",
		"base_url":   "llm.base_url",
		"protocol":   "llm.protocol",
		"model_name": "llm.model_name",
		"data_dir":   "data_dir",
		"book_dir":   "playbooks_dir",
	}
)

// Config 应用程序配置
type Config struct {
	LLM          llm.ClientConfig `mapstructure:"llm"`
	PlaybooksDir string           `mapstructure:"playbooks_dir"`
	DataDir      string           `mapstructure:"data_dir"`
}

// LLMConfig LLM相关配置
type LLMConfig struct {
	APIKey     string `mapstructure:"api_key"`
	BaseURL    string `mapstructure:"base_url"`
	ModelName  string `mapstructure:"model_name"`
	Protocol   string `mapstructure:"protocol"`
	MaxRetries int    `mapstructure:"max_retries"`
}

// NewConfig 创建配置文件创建实例
func NewConfig() *Config {
	configDir := GetConfigDir()

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Error("Failed to create config directory", zap.Error(err))
	}

	// 确保playbooks目录存在
	playbooksDir := viper.GetString("playbooks_dir")
	if playbooksDir == "" {
		playbooksDir = filepath.Join(configDir, "playbooks")
	}

	if err := os.MkdirAll(playbooksDir, 0755); err != nil {
		logger.Error("Failed to create playbooks directory", zap.Error(err))
	}

	// 确保数据目录存在
	dataDir := viper.GetString("data_dir")
	if dataDir == "" {
		dataDir = filepath.Join(configDir, "data")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Error("Failed to create data directory", zap.Error(err))
	}

	return &Config{
		LLM: llm.ClientConfig{
			APIKey:     viper.GetString("llm.api_key"),
			BaseURL:    viper.GetString("llm.base_url"),
			ModelName:  viper.GetString("llm.model_name"),
			Protocol:   viper.GetString("llm.protocol"),
			MaxRetries: viper.GetInt("llm.max_retries"),
		},
		PlaybooksDir: playbooksDir,
		DataDir:      dataDir,
	}
}

// LoadConfig 从配置文件加载配置
func LoadConfig() (*Config, error) {
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，则返回带默认值的配置
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return NewConfig(), nil
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 返回配置实例
	return NewConfig(), nil
}

// GetConfigDir 获取配置目录路径
func GetConfigDir() string {
	configDir := os.Getenv("HOME")
	if configDir == "" {
		configDir = "."
	}
	configDir = filepath.Join(configDir, ".xdiag")

	// 展开环境变量
	return os.ExpandEnv(configDir)
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.yaml")
}

// EnsureConfigDir 确保配置目录存在
func EnsureConfigDir() error {
	configDir := GetConfigDir()
	return os.MkdirAll(configDir, 0755)
}

// SaveModelConfig 保存模型配置
func SaveModelConfig(apiKey, baseURL, protocol, modelName string) error {
	// 确保配置目录存在
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("创建配置目录失败：%v", err)
	}

	// 设置配置
	viper.Set("llm.api_key", apiKey)
	viper.Set("llm.base_url", baseURL)
	viper.Set("llm.protocol", protocol)
	viper.Set("llm.model_name", modelName)

	// 写入配置文件
	configPath := GetConfigPath()
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("保存配置失败：%v", err)
	}

	return nil
}

// SetConfigValue 设置单个配置项
func SetConfigValue(key, value string) error {
	fullKey, ok := keyMap[key]
	if !ok {
		return fmt.Errorf("未知配置项：%s", key)
	}

	// 加载现有配置
	configPath := GetConfigPath()
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，创建一个新的
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 确保配置目录存在
			if err := EnsureConfigDir(); err != nil {
				return fmt.Errorf("创建配置目录失败：%v", err)
			}

			// 设置默认值
			viper.Set("llm.api_key", "")
			viper.Set("llm.base_url", "https://api.openai.com/v1")
			viper.Set("llm.protocol", "openai")
			viper.Set("llm.model_name", "")
			viper.Set("data_dir", "/root/.github.com/bigWhiteXie/xdiag/data")
			viper.Set("playbooks_dir", "/root/.github.com/bigWhiteXie/xdiag/playbooks")

		} else {
			return fmt.Errorf("读取配置失败：%v", err)
		}
	}

	// 设置新值
	viper.Set(fullKey, value)

	// 写入配置文件
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("保存配置失败：%v", err)
	}

	return nil
}

// UnsetConfigValue 删除配置项
func UnsetConfigValue(key string) error {
	fullKey, ok := keyMap[key]
	if !ok {
		return fmt.Errorf("未知配置项：%s", key)
	}

	// 读取配置文件内容
	configPath := GetConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败：%v", err)
	}

	// 解析 YAML
	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("解析配置文件失败：%v", err)
	}

	// 从 map 中删除键
	keys := strings.Split(fullKey, ".")
	current := config
	for i, k := range keys {
		if i == len(keys)-1 {
			delete(current, k)
			break
		}
		if next, ok := current[k].(map[string]interface{}); ok {
			current = next
		} else {
			return fmt.Errorf("配置路径不存在：%s", fullKey)
		}
	}

	// 写回配置文件
	newData, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("序列化配置失败：%v", err)
	}

	err = os.WriteFile(configPath, newData, 0644)
	if err != nil {
		return fmt.Errorf("写入配置文件失败：%v", err)
	}

	// 重新加载配置到 viper
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("重新加载配置失败：%v", err)
	}

	return nil
}
