package config

import (
	"fmt"
	"os"
	"strconv"
)

func GetEnvironment(key string) (string, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	if value == "" {
		return "", fmt.Errorf("environment variable %s is set but empty", key)
	}
	return value, nil
}

func GetEnvironmentInt(key string) (int, error) {
	value, err := GetEnvironment(key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(value)
}
