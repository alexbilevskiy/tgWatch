package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

var Config = ConfigFileStruct{}

func InitConfiguration() {
	UnmarshalJsonFile("config.json", &Config)
}

func UnmarshalJsonFile(path string, dest interface{}) {

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatalf("json file (%s) does not exist: %s", path, err.Error())

		return
	}

	if jsonFile, err := os.Open(path); err != nil {
		log.Fatalf("failed to open json file: %s", err.Error())

		return
	} else {
		defer jsonFile.Close()

		if byteValue, err := ioutil.ReadAll(jsonFile); err != nil {
			log.Fatalf("failed to read json file: %s", err.Error())

			return
		} else {
			if err := json.Unmarshal(byteValue, &dest); err != nil {
				log.Fatalf("failed to parse json file %s: %s", path, err.Error())

				return
			}
		}
	}
}
