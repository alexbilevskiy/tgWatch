package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

var Config = ConfigFileStruct{}

func InitConfiguration() {
	var path string

	path = "config.json"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatal("config file does not exist: " + err.Error())

		return
	}

	if jsonFile, err := os.Open(path); err != nil {
		log.Fatal("failed to open config file: " + err.Error())

		return
	} else {
		defer jsonFile.Close()

		if byteValue, err := ioutil.ReadAll(jsonFile); err != nil {
			log.Fatal("failed to read config file: " + err.Error())

			return
		} else {
			if err := json.Unmarshal(byteValue, &Config); err != nil {
				log.Fatal("failed to parse config file: " + err.Error())

				return
			}
		}
	}
}