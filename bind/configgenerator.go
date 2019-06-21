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
	GeneratedAt string
	MasterIPs   []string
	Zones       []templateZone
}

func GenerateConfigs(zones []string, masterConfig config.NbbxConfig) {
	templateString, err := ioutil.ReadFile("./templates/config.tmpl")
	if err != nil {
		panic(err)
	}
	configTemplate := template.Must(template.New("config").Parse(string(templateString)))
	util.CleanDirectory("./bind-config")

	templateVars := configTemplateVars{
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	for _, currentMaster := range masterConfig.Masters {
		templateVars.ServerName = currentMaster.Name
		templateZones := []templateZone{}

		for _, zonesMaster := range masterConfig.Masters {
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

		var masterIPsWithoutCurrent = make([]string, 0, len(masterConfig.Masters)-1)
		for _, master := range masterConfig.Masters {
			if master.IP != currentMaster.IP {
				masterIPsWithoutCurrent = append(masterIPsWithoutCurrent, master.IP)
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
