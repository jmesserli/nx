package config

import (
	"encoding/json"
	"io/ioutil"
)

type NetboxConfig struct {
	URL    string `json:"url"`
	ApiKey string `json:"api_key"`
}

type ZoneInclude struct {
	Zone         string   `json:"zone"`
	IncludeFiles []string `json:"include_files"`
}

type AdditionalSlavesConfig = map[string][]string

type MasterConfig struct {
	Name             string                 `json:"name"`
	IP               string                 `json:"ip"`
	DottedEmail      string                 `json:"dotted_mail"`
	Zones            []string               `json:"zones"`
	Includes         []ZoneInclude          `json:"includes"`
	AdditionalSlaves AdditionalSlavesConfig `json:"additional_slaves"`
}

type DNSNamespaceConfig struct {
	Masters []MasterConfig `json:"masters"`
}

type NamespaceConfig struct {
	DNS DNSNamespaceConfig `json:"dns"`
}

type NXConfig struct {
	Netbox       NetboxConfig    `json:"netbox"`
	Namespaces   NamespaceConfig `json:"namespaces"`
	UpdatedFiles []string        `json:"-"`
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
