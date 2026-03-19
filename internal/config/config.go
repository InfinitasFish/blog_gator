package config

import (
	//"fmt"
	"os"
	"encoding/json"
)

const configFileName = "/.gatorconfig.json"

type Config struct {
	DbUrl *string `json:"db_url"`
	UserName *string `json:"current_user_name"`
}

// reads config from home directory
func Read() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	fullPath := homeDir + configFileName
	data, err := os.ReadFile(fullPath)
	
	config := Config{}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

// sets current_user_name into Config
// then writes Config into json file
func SetUser(username string, config *Config) error {
	config.UserName = &username
	configJson, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	fullPath := homeDir + configFileName
	err = os.WriteFile(fullPath, configJson, 0666)
	if err != nil {
		return err
	}

	return nil
}
