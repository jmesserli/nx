package bind

import (
	"fmt"
	"io/ioutil"
	"os"
	"text/template"
	"time"

	"github.com/jmesserli/netbox-to-bind/util"

	"github.com/jmesserli/netbox-to-bind/config"
)

type zoneType string

const (
	slave  zoneType = "slave"
	master          = "master"
)

type templateZone struct {
	Name     string
	Type     zoneType
	IsSlave  bool
	MasterIP string
}

type configTemplateVars struct {
	ServerName  string
	GeneratedAt string
	MasterIPs   []string
	Zones       []templateZone
}

func GenerateConfigs(zones []string, zoneConfig config.NbbxConfig) {
	var mastersSet = make(map[config.ZoneMaster]struct{})
	for _, zone := range zoneConfig.Zones {
		if _, ok := mastersSet[zone.Master]; !ok {
			mastersSet[zone.Master] = struct{}{}
		}
	}
	var masterIps = make([]string, 0, len(mastersSet))
	for master := range mastersSet {
		masterIps = append(masterIps, master.IP)
	}

	templateString, err := ioutil.ReadFile("./templates/config.tmpl")
	if err != nil {
		panic(err)
	}
	configTemplate := template.Must(template.New("config").Parse(string(templateString)))
	util.CleanDirectory("./bind-config")

	templateVars := configTemplateVars{
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	for currentMaster := range mastersSet {
		templateVars.ServerName = currentMaster.Name
		templateZones := []templateZone{}

		for _, zone := range zones {
			var foundZone *config.ZoneConfig
			for _, configZone := range zoneConfig.Zones {
				if configZone.Name == zone {
					foundZone = &configZone
					break
				}
			}

			if foundZone == nil {
				continue
			}

			isMaster := foundZone.Master.Name == currentMaster.Name
			var zoneType zoneType
			if isMaster {
				zoneType = master
			} else {
				zoneType = slave
			}
			templateZones = append(templateZones, templateZone{
				Name:     zone,
				IsSlave:  !isMaster,
				Type:     zoneType,
				MasterIP: foundZone.Master.IP,
			})
		}
		templateVars.Zones = templateZones

		var masterIPsWithoutCurrent = make([]string, 0, len(masterIps)-1)
		for _, masterIP := range masterIps {
			if masterIP != currentMaster.IP {
				masterIPsWithoutCurrent = append(masterIPsWithoutCurrent, masterIP)
			}
		}
		templateVars.MasterIPs = masterIPsWithoutCurrent

		f, err := os.Create(fmt.Sprintf("./bind-config/%s.conf", currentMaster.Name))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		configTemplate.Execute(f, templateVars)
	}

}
