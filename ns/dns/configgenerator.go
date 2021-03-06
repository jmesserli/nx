package dns

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"text/template"
	"time"

	"peg.nu/nx/cache"
	"peg.nu/nx/config"
	"peg.nu/nx/util"
)

type zoneType string

const (
	slave  zoneType = "slave"
	master zoneType = "master"
)

type templateZone struct {
	Name     string
	Type     zoneType
	IsSlave  bool
	MasterIP string
}

type configTemplateVars struct {
	ServerName  string
	ServerIP    string
	GeneratedAt string
	MasterIPs   []string
	Zones       []templateZone
}

func GenerateConfigs(zones []string, conf *config.NXConfig) {
	templateString, err := ioutil.ReadFile("templates/bind-config.tmpl")
	if err != nil {
		panic(err)
	}
	configTemplate := template.Must(template.New("config").Parse(string(templateString)))
	cw := cache.New("generated/hashes/bind-config.json")
	ignoreRegex := regexp.MustCompile("(?m)^ \\* Generated at.*$")

	templateVars := configTemplateVars{
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	for _, currentMaster := range conf.Namespaces.DNS.Masters {
		templateVars.ServerName = currentMaster.Name
		templateVars.ServerIP = currentMaster.IP
		templateZones := []templateZone{}

		for _, zonesMaster := range conf.Namespaces.DNS.Masters {
			isMaster := zonesMaster.Name == currentMaster.Name
			masterZoneType := master
			if !isMaster {
				masterZoneType = slave
			}

			for _, zone := range zonesMaster.Zones {
				if !util.SliceContainsString(zones, zone) {
					continue
				}

				templateZones = append(templateZones, templateZone{
					IsSlave:  !isMaster,
					MasterIP: zonesMaster.IP,
					Name:     zone,
					Type:     masterZoneType,
				})
			}
		}

		templateVars.Zones = templateZones

		var masterIPsWithoutCurrent = make([]string, 0, len(conf.Namespaces.DNS.Masters)-1)
		for _, master := range conf.Namespaces.DNS.Masters {
			if master.IP != currentMaster.IP {
				masterIPsWithoutCurrent = append(masterIPsWithoutCurrent, master.IP)
			}
		}
		templateVars.MasterIPs = masterIPsWithoutCurrent

		_, err = cw.WriteTemplate(
			fmt.Sprintf("generated/bind-config/%s.conf", currentMaster.Name),
			configTemplate,
			templateVars,
			[]*regexp.Regexp{ignoreRegex},
			false,
		)
		if err != nil {
			panic(err)
		}
	}

	util.CleanDirectoryExcept("generated/bind-config", cw.ProcessedFiles, conf)
	conf.UpdatedFiles = append(conf.UpdatedFiles, cw.UpdatedFiles...)
}
