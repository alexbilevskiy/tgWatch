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
		log.Fatal(err)

		return
	}

	if jsonFile, err := os.Open(path); err != nil {
		log.Fatal(err)

		return
	} else {
		defer jsonFile.Close()

		if byteValue, err := ioutil.ReadAll(jsonFile); err != nil {
			log.Fatal(err)

			return
		} else {
			if err := json.Unmarshal(byteValue, &Config); err != nil {
				log.Fatal(err)

				return
			}
		}
	}
}