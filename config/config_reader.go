package config

import (
	"encoding/json"
	"io/ioutil"
)

type NetboxConfig struct {
	URL    string `json:"url"`
	ApiKey string `json:"api_key"`
}

type MasterConfig struct {
	Name        string   `json:"name"`
	IP          string   `json:"ip"`
	DottedEmail string   `json:"dotted_mail"`
	Zones       []string `json:"zones"`
}

type DNSNamespaceConfig struct {
	Masters []MasterConfig `json:"masters"`
}

type NamespaceConfig struct {
	DNS DNSNamespaceConfig `json:"dns"`
}

type NXConfig struct {
	Netbox     NetboxConfig    `json:"netbox"`
	Namespaces NamespaceConfig `json:"namespaces"`
}

func ReadConfig(path string) NXConfig {
	fileContent, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	config := NXConfig{}
	err = json.Unmarshal(fileContent, &config)
	if err != nil {
		panic(err)
	}

	return config
}