package utils

import (
	"encoding/json"
	"fmt"
)

// UnmarshalMap 将 map[string]interface{} 反序列化为指定的结构体
func UnmarshalMap(data map[string]interface{}, target interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal map: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target: %w", err)
	}

	return nil
}
