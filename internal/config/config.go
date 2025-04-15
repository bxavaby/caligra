// BYZRA ⸻ internal/config/config.go
// config loading & management

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// config for daemon mode
type DaemonConfig struct {
	Watch struct {
		Paths []string `toml:"paths"`
	} `toml:"watch"`
	Filter struct {
		Extensions []string `toml:"extensions"`
	} `toml:"filter"`
}

// loads the daemon config
func LoadDaemonConfig() (*DaemonConfig, error) {
	// search common locations
	paths := []string{
		"config/scroud.toml",
		"./scroud.toml",
		filepath.Join(os.Getenv("HOME"), ".caligra/config/scroud.toml"),
	}

	var configPath string
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return nil, fmt.Errorf("scroud.toml not found in search paths")
	}

	var config DaemonConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// filter out commented paths
	var activePaths []string
	for _, path := range config.Watch.Paths {
		if len(path) > 0 && path[0] != '#' {
			activePaths = append(activePaths, path)
		}
	}
	config.Watch.Paths = activePaths

	return &config, nil
}

// returns default config values
func GetDefaultConfig() *DaemonConfig {
	config := &DaemonConfig{}
	config.Watch.Paths = []string{
		os.Getenv("HOME") + "/Downloads",
	}
	config.Filter.Extensions = []string{
		".jpg", ".jpeg", ".png", ".gif",
		".mp3", ".flac", ".opus", ".ogg",
		".mp4", ".avi",
		".txt", ".md", ".html",
	}
	return config
}

// saves the current configuration to a file
func SaveDaemonConfig(config *DaemonConfig, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Open file for writing
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(config)
}

// config directory exists
func SetupConfigDir() (string, error) {
	configDir := filepath.Join(os.Getenv("HOME"), ".caligra/config")
	err := os.MkdirAll(configDir, 0755)
	return configDir, err
}
