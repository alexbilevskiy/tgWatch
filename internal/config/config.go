package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Config struct {
	ApiId     int32             `json:"ApiId"`
	ApiHash   string            `json:"ApiHash"`
	WebListen string            `json:"WebListen"`
	Mongo     map[string]string `json:"Mongo"`
	Debug     bool              `json:"Debug"`
	TDataDir  string            `json:"TDataDir"`
}

func InitConfiguration() (*Config, error) {
	var cfg = Config{}
	err := UnmarshalJsonFile("config.json", &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}

func UnmarshalJsonFile(path string, dest interface{}) error {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("json file does not exist: %w", err.Error())
	}

	if jsonFile, err := os.Open(path); err != nil {
		return fmt.Errorf("failed to open json file: %w", err)
	} else {
		defer jsonFile.Close()

		if byteValue, err := ioutil.ReadAll(jsonFile); err != nil {
			return fmt.Errorf("failed to read json file: %w", err)
		} else {
			if err := json.Unmarshal(byteValue, &dest); err != nil {
				return fmt.Errorf("failed to parse json file: %w", err)
			}
		}
	}

	return nil
}
