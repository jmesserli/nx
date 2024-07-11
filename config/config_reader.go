package config

import (
	"encoding/json"
	"os"
)

type NetboxConfig struct {
	URL    string `json:"url"`
	ApiKey string `json:"api_key"`
}

type ZoneInclude struct {
	Zone         string   `json:"zone"`
	IncludeFiles []string `json:"include_files"`
}

type AdditionalSecondariesConfig = map[string][]string

type PrimaryConfig struct {
	Name                  string                      `json:"name"`
	IP                    string                      `json:"ip"`
	Port                  int                         `json:"port"`
	DottedEmail           string                      `json:"dotted_mail"`
	Zones                 []string                    `json:"zones"`
	DnssecZones           []string                    `json:"dnssec_zones"`
	Includes              []ZoneInclude               `json:"includes"`
	AdditionalSecondaries AdditionalSecondariesConfig `json:"additional_slaves"`
}

type DNSNamespaceConfig struct {
	Primaries []PrimaryConfig `json:"masters"`
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
	fileContent, err := os.ReadFile(path)
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
