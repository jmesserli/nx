package config

import (
	"encoding/json"
	"io/ioutil"
)

type MasterConfig struct {
	Name        string   `json:"name"`
	IP          string   `json:"ip"`
	DottedEmail string   `json:"dotted_email"`
	Zones       []string `json:"zones"`
}

type NbbxConfig struct {
	Masters []MasterConfig `json:"masters"`
}

func ReadConfig(path string) NbbxConfig {
	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	config := NbbxConfig{}
	err = json.Unmarshal(fileContent, &config)
	if err != nil {
		panic(err)
	}

	return config
}
