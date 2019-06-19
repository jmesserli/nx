package config

import (
	"encoding/json"
	"io/ioutil"
)

type ZoneMaster struct {
	IP   string `json:"ip"`
	Name string `json:"name"`
}

type ZoneConfig struct {
	Name       string     `json:"name"`
	Master     ZoneMaster `json:"master"`
	DottedMail string     `json:"dotted_mail"`
}

type NbbxConfig struct {
	Zones []ZoneConfig `json:"zones"`
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
