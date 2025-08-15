package entities

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type ConfigEntity struct {
	LlmKey string `json:"llm_key,omitempty"`
}

func (c *ConfigEntity) GetConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			configDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	case "darwin":
		configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support")
	default: // linux and other unix-like systems
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(os.Getenv("HOME"), ".config")
		}
	}

	gommitConfigDir := filepath.Join(configDir, "gommit")
	err := os.MkdirAll(gommitConfigDir, 0755)
	if err != nil {
		return "", err
	}

	return gommitConfigDir, nil
}

func (c *ConfigEntity) GetConfigFilePath() (string, error) {
	configDir, err := c.GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.json"), nil
}

func (c *ConfigEntity) Load() error {
	configPath, err := c.GetConfigFilePath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, c)
}

func (c *ConfigEntity) Save() error {
	configPath, err := c.GetConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}