package config

import (
	"fmt"
	"os"
	"strconv"
)

func GetEnvironment(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", key)
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
